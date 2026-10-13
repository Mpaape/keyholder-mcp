# Keyholder MCP: especificação do cofre de credenciais via MCP para agentes de IA

Versão 0.1, 2026-10-13. Combina a arquitetura sobre o OpenBao com o modelo de produto: referências, sessões e adaptadores. Itens não verificados estão marcados **[a verificar]**.

## 1. Objetivo

Eliminar a cópia manual de senhas, tokens, chaves e URLs para chats, repositórios, arquivos `.env` e comandos escritos pelo modelo. O agente vê nomes e metadados, nunca valores. A resolução ocorre só na execução (late binding), e o acesso expira.

O MCP é interface de descoberta e invocação. Quem protege e injeta os segredos é o cofre e o runner.

## 2. Limites honestos

- Não protege contra um agente deliberadamente hostil que roda com o seu usuário do sistema.
- "Não expor à LLM" não significa "nenhum processo acessa o segredo": o processo executado recebe o valor. Cada modo de execução documenta o seu nível de garantia (seção 7).
- Mascaramento de saída é defensivo, não garantia matemática: um valor codificado escapa.
- Fora de escopo: administrar IAM da AWS ou do GCP, uso em equipe, monetização.

## 3. Decisão de base (a confirmar)

Apoiar-se no OpenBao em vez de criar cofre e daemon do zero. Motivos, todos verificados na documentação oficial do OpenBao:

- Token com TTL, limite rígido (`-explicit-max-ttl`) e limite de usos (`-use-limit`).
- Response wrapping: token de uso único, com TTL próprio.
- Leases com revogação automática; revogar um token revoga os leases criados com ele.
- Login MFA com TOTP embutido (gratuito), além de Duo, Okta e PingID (serviços de terceiros).
- AWS e GCP existem como plugins externos oficiais (`openbao-plugins`), instalados por bloco `plugin` com imagem fixada por digest. Não são embutidos.
- Binários para Windows, macOS e Linux; imagens de contêiner; Web UI.

Alternativa: KeePassXC + [`kdbx`](https://github.com/yarrasys/kdbx) (MIT). Sem servidor, mas sem expiração de acesso documentada.

## 4. Arquitetura

```
Cliente de IA local --tools--> MCP do cofre (novo) --HTTP--> OpenBao
                                     |                          ^
                                  executa                  emite token
                                     v                     (você, com TOTP)
                              Comando permitido
                              (variáveis injetadas)
                                     |
                              AWS, GCP, LLMs
```

O MCP é o único componente novo. Ele guarda o token só em memória e nunca devolve valores.

## 5. Requisitos

| # | Requisito | Origem |
| --- | --- | --- |
| R1 | Apoiar-se em cofre open source existente | Pedido |
| R2 | MCP local que entrega credenciais às IAs locais | Pedido |
| R3 | Variáveis de ambiente simples (AWS, GCP, chaves de LLM) | Pedido |
| R4 | Valores não aparecem no chat | Pedido |
| R5 | Interface gráfica no servidor | Pedido |
| R6 | Windows, Linux e macOS | Pedido |
| R7 | Servidor acessível ao WSL | Pedido |
| R8 | Acesso temporário: token com prazo e segundo fator gratuito | Pedido |
| R9 | Referências `vault://workspace/projeto/nome` e placeholders `{{...}}` em comandos | Spec de produto |
| R10 | Workspace, projeto e ambiente como caminhos e políticas do OpenBao | Spec de produto |
| R11 | Sessão por projeto e por cliente, renovável (24 h ou 72 h) | Spec de produto |
| R12 | Substituição segura de argumentos, sem concatenar em shell | Spec de produto |
| R13 | Auditoria sem valores | Proposta |
| R14 | Contribuir para um projeto upstream | Pedido |

## 6. Superfície MCP

Nomes e ações, nunca valores. Não existe tool que devolva o valor em claro.

| Tool | O que faz | Retorna |
| --- | --- | --- |
| `cofre_listar` | Lista referências do projeto atual | Nomes, tipos, descrições |
| `cofre_status` | Estado do cofre e da sessão | Selado ou aberto, TTL restante |
| `cofre_executar` | Roda um comando permitido, resolvendo `vault://` e `{{...}}` na execução | Saída mascarada e código de saída |
| `cofre_solicitar_acesso` | Pede acesso temporário ao humano | Pendente, aprovado ou negado; nunca o token |

Exemplo de chamada planejada pelo agente:

```
curl -H 'Authorization: Bearer {{vault://meu-projeto/dev/API_TOKEN}}' https://api.exemplo.com/health
```

## 7. Modos de execução e garantias

- **Modo A, runner (preferencial):** o agente envia comando com referências; outro processo resolve e executa. O valor não passa pelo diálogo com a LLM.
- **Modo B, ferramentas nativas:** AWS CLI, gcloud ou ssh-agent usam mecanismos do próprio ecossistema. O valor pode existir dentro do processo da ferramenta e aparecer em depuração. Sem promessa de isolamento absoluto. **[a verificar]** como o `gcloud` autentica sem o agente receber token ou chave.
- **Modo C, navegador:** preenchimento de campos por componente local. O sigilo depende de o agente conseguir ler o DOM depois. Fora do MVP.

Uma integração sem late binding seguro deve declarar a limitação em vez de fingir sigilo.

## 8. Fluxo de acesso temporário

1. Você faz login no OpenBao com TOTP.
2. Emite um token filho com política mínima, TTL, limite de usos e `wrap-ttl`.
3. O token chega ao MCP embrulhado (uso único), por canal local fora do chat.
4. O MCP desembrulha e guarda o token só em memória.
5. `cofre_executar` lê o segredo ou pede credencial dinâmica e injeta no comando.
6. O token expira. **[a verificar]** se a expiração revoga os leases criados com ele (a documentação afirma a revogação explícita).

## 9. WSL e multiplataforma

Recomendação: servidor dentro do WSL ou em contêiner, com o Windows acessando por `localhost`.

Motivo: no modo mirrored, o handshake TCP de WSL para o Windows via `127.0.0.1` tem falha reportada e reproduzida no WSL 2.7.14.0 ([issue 40343](https://github.com/microsoft/WSL/issues/40343)). **[a verificar]** se foi corrigida. No modo NAT, o WSL acessa o Windows pelo IP do host, o que exige o servidor aceitar conexões além do loopback.

O modo `bao server -dev` serve só para prova de conceito: memória, 127.0.0.1:8200, sem TLS.

## 10. Modelo de ameaças

| Ameaça | Mitigação | Limite |
| --- | --- | --- |
| Valor no chat ou logs | MCP devolve nomes; saída mascarada | Mascaramento casa só valor exato |
| Credencial permanente vazada | Token com TTL e limite rígido; credenciais dinâmicas com lease | KV não emite leases |
| Token interceptado na entrega | Response wrapping | Só protege a entrega |
| Agente com seu usuário alcança o token | Token só em memória do MCP | Sem processo separado, um agente determinado derrota a proteção |
| Força bruta no TOTP | 5 falhas seguidas bloqueiam; exigir OpenBao 2.3.2 ou superior (CVE-2025-55003, **[a verificar]** no advisory) | MFA embutido não protege por padrão; aplicar rate limit |
| Segredos paginados em disco | Desativar ou criptografar swap | Depende do sistema |

## 11. Decisões em aberto

1. OpenBao ou `kdbx` como cofre (define o projeto upstream).
2. Onde fica o servidor: WSL ou contêiner (recomendado) ou Windows.
3. Como destravar após reiniciar: Shamir manual ou auto unseal.
4. Segundo fator: TOTP nativo (gratuito) ou serviço externo.
5. Credencial do GCP: chave JSON de service account ou token OAuth dinâmico.
6. Escrita da IA no cofre: caminho dedicado ou nenhuma.
7. Canal local de entrega do token ao MCP, fora do chat.
8. Licença do projeto.
9. Linguagem do MCP e do runner.

## 12. Plano de entrega

1. **Prova de conceito:** `bao server -dev` no WSL, chaves falsas no KV, token com `-ttl`, `-use-limit` e `-wrap-ttl`, acessado do Windows. Critério: funciona nos dois lados.
2. **MCP mínimo:** `cofre_listar`, `cofre_status`, `cofre_executar` com mascaramento e lista de comandos permitidos. Critério: nenhum valor na saída.
3. **Acesso temporário:** emissão com TOTP e token embrulhado fora do chat. Critério: o token expira e o acesso para.
4. **Credenciais dinâmicas:** plugins de AWS e GCP ligados às variáveis de ambiente. Critério: a credencial expira e é revogada.
5. **Servidor persistente:** armazenamento, TLS e política de unseal.
6. **SSH e navegador:** adaptadores posteriores.
7. **Upstream:** ler o `CONTRIBUTING.md` do projeto escolhido e abrir issue propondo "executar com variáveis injetadas" e acesso temporário.

## 13. Critérios de aceite

- CA-01: o token cadastrado não aparece no chat, na descrição da tool nem na resposta MCP.
- CA-02: chamada real de API com referência; o servidor recebe o token certo e o agente só o resultado.
- CA-03: `aws sts get-caller-identity` e `gcloud projects list` funcionam sem o agente escrever chaves.
- CA-04: abrir um projeto mostra só as referências do escopo.
- CA-05: dentro da sessão, repetir operações sem reaprovação; após expirar, exige desbloqueio fora do chat.
- CA-06: logs, mensagens MCP e artefatos do produto não contêm valores.
- CA-07: integração sem late binding seguro declara a limitação.
- CA-08: funciona com pelo menos dois clientes MCP sem mudar o armazenamento.

## 14. Fontes

- OpenBao: [token create](https://openbao.org/docs/commands/token/create/), [response wrapping](https://openbao.org/docs/concepts/response-wrapping/), [login MFA](https://openbao.org/docs/auth/login-mfa/), [lease](https://openbao.org/docs/concepts/lease/), [plugins](https://openbao.org/docs/plugins/), [seal](https://openbao.org/docs/concepts/seal/), [dev server](https://openbao.org/docs/concepts/dev-server/), [instalação](https://openbao.org/docs/install/)
- [openbao-plugins](https://github.com/openbao/openbao-plugins)
- [kdbx](https://github.com/yarrasys/kdbx)
- [WSL: rede](https://learn.microsoft.com/en-us/windows/wsl/networking), [issue 40343](https://github.com/microsoft/WSL/issues/40343)
