# Especificação

Versão 0.3, 2026-10-13. Itens ainda não verificados estão marcados **[a verificar]**.

## 1. Objetivo

Acabar com a cópia manual de senhas, tokens, chaves e URLs para chats, repositórios, arquivos `.env` e comandos escritos por um modelo. O agente vê nomes e metadados, nunca valores. A resolução só acontece na hora da execução (late binding), e o acesso expira.

O MCP é a interface de descoberta e chamada. Quem protege e injeta os segredos é o cofre e o runner.

## 2. Limites honestos

- Não protege contra um agente deliberadamente hostil rodando com o seu usuário do sistema.
- "Não exposto ao modelo" não quer dizer "nenhum processo toca o segredo": o processo executado recebe o valor. Cada modo de execução declara a sua garantia ([Modelo de segurança](security.md#modos-de-execucao)).
- O mascaramento da saída é uma defesa, não uma garantia matemática: um valor codificado escapa.
- Fora do escopo: gerenciar IAM da AWS ou do GCP, uso em equipe, monetização.

## 3. Decisão de base

Construir sobre o OpenBao em vez de escrever cofre e daemon do zero. Cada motivo abaixo está verificado na documentação oficial do OpenBao:

- Tokens com TTL, máximo rígido (`-explicit-max-ttl`) e limite de usos (`-use-limit`) ([token create](https://openbao.org/docs/commands/token/create/)).
- Response wrapping: token de uso único com TTL próprio ([response wrapping](https://openbao.org/docs/concepts/response-wrapping/)).
- Leases com revogação automática; revogar um token revoga os leases criados com ele ([lease](https://openbao.org/docs/concepts/lease/)).
- Login MFA com TOTP embutido (gratuito), além de Duo, Okta e PingID (serviços de terceiros) ([login MFA](https://openbao.org/docs/auth/login-mfa/)).
- AWS e GCP existem como plugins oficiais **externos** ([openbao-plugins](https://github.com/openbao/openbao-plugins)), instalados por um bloco `plugin` fixado por digest; não vêm embutidos ([plugins](https://openbao.org/docs/plugins/)).
- Binários para Windows, macOS e Linux; imagens de contêiner; interface web ([instalação](https://openbao.org/docs/install/)).

Alternativa considerada: KeePassXC com [`kdbx`](https://github.com/yarrasys/kdbx) (MIT). Sem servidor, mas sem expiração de acesso documentada.

## 4. Requisitos

| # | Requisito | Situação |
| --- | --- | --- |
| R1 | Usar um cofre open source existente | Feito (OpenBao) |
| R2 | Servidor MCP local que entrega credenciais a agentes de IA locais | Feito |
| R3 | Variáveis de ambiente simples (AWS, GCP, chaves de LLM) | Feito: valores estáticos, e credenciais dinâmicas de AWS e GCP via `credentials` |
| R4 | Valores nunca aparecem no chat | Feito |
| R5 | Interface gráfica no servidor | Feito (interface web do OpenBao) |
| R6 | Windows, Linux e macOS | Feito (CI nos três) |
| R7 | Servidor acessível a partir do WSL | Documentado ([Windows e WSL](windows-wsl.md)) |
| R8 | Acesso temporário: token com expiração e segundo fator gratuito | Feito (token embrulhado + TTL; TOTP no OpenBao) |
| R9 | Referências `vault://caminho/CHAVE` e placeholders `{{...}}` em comandos | Feito |
| R10 | Workspace, projeto e ambiente como caminhos e políticas do OpenBao | Feito (convenção de caminhos + escopo) |
| R11 | Sessão por projeto e cliente, renovável (24 h ou 72 h) | Em parte: TTL por sessão; renovar é emitir um novo token embrulhado |
| R12 | Substituição segura de argumentos, sem concatenar em shell | Feito |
| R13 | Auditoria sem valores | Feito |
| R14 | Contribuir com um projeto upstream | Planejado |

## 5. Superfície MCP

Nomes e ações, nunca valores. Nenhuma tool devolve um valor em claro.

| Tool | Faz | Devolve | Situação |
| --- | --- | --- | --- |
| `keyholder_status` | Estado do cofre e da sessão | Selado ou não, TTL restante, políticas | Implementada |
| `keyholder_list` | Lista as referências no escopo do projeto | Nomes `vault://` e versões | Implementada |
| `keyholder_run` | Roda um comando permitido, resolvendo `{{vault://...}}` em `args` e `env` na hora da execução | Saída mascarada, código de saída, referências usadas | Implementada |
| `keyholder_request_access` | Pede acesso temporário ao humano | Pendente, aprovado ou negado; nunca o token | Planejada |

Referência campo a campo: [Configuração](configuration.md#tools).

## 6. Modos de execução

- **Modo A, runner (preferido):** o agente envia um comando com referências; outro processo resolve e executa. O valor não passa pela conversa.
- **Modo B, ferramentas nativas:** AWS CLI, gcloud ou ssh-agent usam os próprios mecanismos. O valor pode ficar no processo da ferramenta e aparecer em saída de depuração; não se promete isolamento absoluto. Verificado: o `gcloud` lê um token de acesso do arquivo indicado em `CLOUDSDK_AUTH_ACCESS_TOKEN_FILE`, que o runner escreve para a execução e apaga depois ([tutorial de nuvem](tutorials/cloud.md)).
- **Modo C, navegador:** um componente local preenche um campo. O sigilo depende de o agente conseguir ler a página depois. Fora do MVP.

Uma integração sem late binding seguro precisa declarar o limite em vez de fingir sigilo.

## 7. Fluxo de acesso temporário

1. Você entra no OpenBao com TOTP.
2. Você emite um token filho com política mínima, TTL, limite de usos opcional e `wrap-ttl`.
3. O token embrulhado chega ao servidor pelo ambiente do cliente MCP, fora do chat.
4. O servidor o desembrulha e guarda o token só em memória.
5. O `keyholder_run` lê o segredo e o injeta no comando.
6. O token expira; as leituras devolvem 403 e as tools dizem ao agente para pedir renovação. **[a verificar]** se a expiração (não a revogação explícita) também revoga os leases criados com o token; a documentação afirma isso para a revogação.

## 8. Prova de conceito, verificada

Contra o `bao server -dev` 2.7.1, gravada e reconferida pelos [tutoriais](tutorials/index.md):

- A API de destino recebeu o valor certo de `Authorization: Bearer`, enquanto o agente leu `[hidden:vault://my-project/dev/API_TOKEN]`.
- Nenhum valor guardado e nenhum token do cofre apareceram em resposta MCP alguma.
- Uma referência fora do `KEYHOLDER_SCOPE` e um comando fora do `KEYHOLDER_ALLOW` foram recusados, mesmo com política ampla no cofre.
- Depois do TTL, o `keyholder_status` informou a expiração e o `keyholder_run` falhou com acesso negado.
- Reusar um token embrulhado falhou no desembrulho.
- `BAO_*`, `VAULT_*` e `KEYHOLDER_*` nunca chegam ao processo filho.
- O CI compila e testa em Linux, Windows e macOS.

- Uma credencial dinâmica criada pelo plugin real de AWS do OpenBao rodou a AWS CLI contra um emulador local da AWS (moto) e foi apagada quando o comando terminou.

Ainda não testado: clientes MCP reais de ponta a ponta; o plugin do GCP e contas AWS reais.

## 9. Decisões em aberto

1. Como fazer o unseal depois de reiniciar: Shamir manual ou auto unseal.
2. Segundo fator: TOTP embutido ou serviço externo.
3. Credencial do GCP: chave JSON de service account ou token OAuth dinâmico.
4. Escrita do agente no cofre: um caminho dedicado, ou nenhuma.
5. Entrega do token: hoje `BAO_WRAPPING_TOKEN` no ambiente do cliente, sem valor depois de desembrulhado. Pode vir um prompt local ou um socket.

Decidido: OpenBao como cofre; Go como linguagem; licença MIT; OpenBao dentro do WSL ou de um contêiner no Windows.

## 10. Critérios de aceite

| # | Critério | Situação |
| --- | --- | --- |
| AC-01 | Um token guardado não aparece no chat, na descrição de uma tool nem em resposta MCP | Atendido |
| AC-02 | Chamada real a uma API com referência: o servidor recebe o token certo, o agente só o resultado | Atendido |
| AC-03 | `aws sts get-caller-identity` e `gcloud projects list` funcionam sem o agente escrever chaves | Atendido para AWS contra um emulador, no CI; tratamento do token do gcloud atendido; plugin do GCP não executado |
| AC-04 | Um projeto vê só as referências do seu escopo | Atendido |
| AC-05 | Dentro da sessão, operações repetidas não pedem nova aprovação; depois da expiração, renovação fora do chat | Atendido |
| AC-06 | Logs, mensagens MCP e artefatos do produto não contêm valores | Atendido |
| AC-07 | Uma integração sem late binding seguro declara o seu limite | Atendido (documentado) |
| AC-08 | Funciona com pelo menos dois clientes MCP sem mudar o armazenamento | Aberto (documentado, não roda no CI) |

## 11. Roteiro

1. ~~Prova de conceito e MCP mínimo~~: feito, publicado como v0.1.0.
2. ~~**Credenciais dinâmicas:** plugins de AWS e GCP do OpenBao mapeados para variáveis de ambiente~~: feito na v0.2.0. Credenciais AWS `iam_user` são revogadas depois da execução; STS e tokens do GCP expiram sozinhos.
3. **`keyholder_request_access`:** o agente pede, o humano aprova com TOTP, um novo token embrulhado chega ao servidor.
4. **Guia de servidor persistente:** armazenamento, TLS e unseal.
5. **Adaptadores de SSH e navegador.**
6. **Upstream:** propor "executar com variáveis injetadas" e acesso temporário a um projeto existente.
