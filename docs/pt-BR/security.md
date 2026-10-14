# Modelo de segurança

O Keyholder reduz o alcance de um vazamento de credencial: os valores ficam fora da conversa, o acesso expira, e só comandos nomeados num projeto conseguem usá-los. Ele **não** impede um agente hostil que já roda com o seu usuário do sistema. Esta página diz exatamente onde está essa linha.

## O que é protegido

| Ameaça | Mitigação | Onde é demonstrado |
| --- | --- | --- |
| O valor aparece no chat, na descrição de uma tool ou num resultado | As tools devolvem só nomes e estado; a saída é mascarada | [Início rápido](tutorials/quickstart.md) |
| O valor vai para logs ou erros | A linha de auditoria traz comando, referências e código de saída; erros do cofre viram só um código de status | [Configuração](configuration.md#trilha-de-auditoria) |
| Uma credencial permanente vaza | Token de sessão com TTL e máximo rígido (`-explicit-max-ttl`) | [Acesso temporário](tutorials/access.md) |
| O token é interceptado a caminho do servidor | Response wrapping: uso único, TTL curto; um desembrulho que falha revela a interceptação | [Acesso temporário](tutorials/access.md) |
| O agente alcança segredos de outro projeto | `KEYHOLDER_SCOPE` checado antes de qualquer leitura, mesmo com política ampla no cofre | [Proteções](tutorials/guardrails.md) |
| O agente roda código arbitrário com o segredo | Lista de comandos pelo nome exato; sem shell; sem caminho | [Proteções](tutorials/guardrails.md) |
| Caracteres especiais de um segredo viram código | Os argumentos vão para o `exec` como lista | [Configuração](configuration.md#keyholder_run) |
| O processo filho lê o token do cofre | `BAO_*`, `VAULT_*`, `KEYHOLDER_*` removidos do ambiente dele | [Proteções](tutorials/guardrails.md) |
| O agente redireciona o filho para outro cofre | Nomes de variáveis reservados são recusados em `env` | [Proteções](tutorials/guardrails.md) |
| Um comando que não termina | Timeout e teto de 64 KiB de saída | [Configuração](configuration.md#keyholder_run) |
| Uma chave de nuvem sobrevive à tarefa | Credenciais dinâmicas lidas na execução e revogadas quando o comando termina (usuários IAM da AWS são apagados) | [Credenciais de nuvem](tutorials/cloud.md) |
| O agente escolhe qualquer caminho de engine | Caminhos dinâmicos precisam estar em `KEYHOLDER_DYNAMIC`, conferidos antes da leitura | [Credenciais de nuvem](tutorials/cloud.md) |

## Em relação a outros servidores MCP

Os outros servidores MCP de OpenBao e Vault que encontramos expõem uma tool que lê um segredo para o modelo; o Keyholder não tem nenhuma. Veja a [Comparação](comparison.md).

## Limites conhecidos

Não são bugs; são a borda do que este desenho consegue fazer.

- **O comando executado recebe o valor.** Qualquer uso real de credencial recebe. A garantia é sobre a conversa com o modelo, não sobre o processo que usa o segredo.
- **O mascaramento só casa o valor exato.** Um interpretador liberado consegue codificá-lo e escapar; o [tutorial de mascaramento](tutorials/masking.md) demonstra. Mantenha shells e interpretadores fora de `KEYHOLDER_ALLOW`.
- **Um agente hostil com o seu usuário do sistema consegue contornar a fronteira.** Ele poderia ler a memória do servidor ou iniciar processos fora do MCP. O Keyholder não roda com um usuário separado.
- **Segredos do KV não têm lease.** Um segredo estático vale até você rotacioná-lo; só o token da sessão expira. Use [credenciais dinâmicas](configuration.md#credenciais-dinamicas) onde a nuvem permitir.
- **Nem toda credencial dinâmica pode ser revogada.** Credenciais STS da AWS e tokens de acesso do GCP só expiram (no mínimo 15 minutos no STS, até uma hora nos tokens do GCP). Prefira roles `iam_user` da AWS quando a chave precisa morrer junto com o comando.

## Modos de execução

| Modo | O que é | Garantia |
| --- | --- | --- |
| **A. Runner** (este projeto) | O agente envia um comando com referências; o Keyholder resolve e executa | O valor nunca passa pela conversa |
| B. Ferramentas nativas | AWS CLI, gcloud ou ssh-agent leem credenciais pelos próprios mecanismos | O valor pode ficar no processo da ferramenta e em saída de depuração; não se promete isolamento absoluto |
| C. Navegador | Um componente local preenche um campo de formulário | Depende de o agente conseguir ler a página depois; fora do escopo |

Uma integração sem late binding seguro precisa dizer isso, em vez de fingir que a credencial nunca chegou ao processo.

## Segundo fator

O [Login MFA](https://openbao.org/docs/auth/login-mfa/) do OpenBao suporta TOTP (embutido, gratuito), além de Duo, Okta e PingID (serviços de terceiros). Exija-o no login que você usa para emitir os tokens embrulhados. A documentação do OpenBao avisa que o MFA embutido não impede força bruta no TOTP por padrão e recomenda limites de taxa por cliente em `/sys/mfa/validate`; por padrão ele aceita 5 falhas seguidas de TOTP. Use o OpenBao 2.3.2 ou superior: versões anteriores aceitavam códigos TOTP com espaços (CVE-2025-55003, segundo fontes secundárias).

## Reportar uma vulnerabilidade

Veja o [SECURITY.md](https://github.com/Mpaape/keyholder-mcp/blob/main/SECURITY.md): use o aviso privado de vulnerabilidade do GitHub, não uma issue pública.
