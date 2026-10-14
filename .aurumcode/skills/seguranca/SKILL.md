---
name: seguranca
version: 1
paths: ["**"]
---
Regras de segurança do Keyholder MCP. Cada seção é uma regra citável pelo gate.

## SEG-001 Valor de segredo nunca volta ao agente
severity: error
Nenhuma resposta de tool MCP, conteúdo estruturado, mensagem de erro ou descrição de tool pode conter o valor resolvido de uma referência, o token do cofre ou o token embrulhado. Só nomes `vault://...`, estado e saída mascarada.

## SEG-002 Segredo nunca vai para log, erro ou arquivo
severity: error
Logs (inclusive a linha de auditoria), erros formatados com `%v`/`%w`, panics e arquivos temporários não podem incluir valores resolvidos, argumentos expandidos, variáveis de ambiente do filho nem o token. A auditoria registra comando, referências, código de saída e duração.

## SEG-003 Execução sem shell
severity: error
Comandos rodam com `exec.CommandContext(ctx, caminho, args...)` e argumentos como lista. É proibido montar comando por concatenação, chamar `sh -c`, `bash -c`, `cmd /c`, `powershell -Command` ou interpretar o texto de um argumento.

## SEG-004 Lista de comandos permitidos
severity: error
Todo comando precisa passar por `Runner.Allow` antes de `exec.LookPath`. O nome não pode conter separador de caminho nem placeholder. Mudanças que afrouxem essa checagem, ou que adicionem por padrão interpretadores (`sh`, `bash`, `python`, `node`, `env`), são bloqueantes.

## SEG-005 Token do cofre fora do processo filho
severity: error
O ambiente do filho passa por `filterEnv`, que remove `BAO_*`, `VAULT_*` e `KEYHOLDER_*`, e o pedido não pode definir variáveis com esses prefixos. As variáveis de token são apagadas do processo logo após `bao.Open`.

## SEG-006 Escopo do projeto
severity: error
Toda referência resolvida precisa passar por `ref.InScope` com o escopo configurado. Leitura direta no KV que contorne o runner ou o escopo é bloqueante.

## SEG-007 Validação de referências
severity: error
`ref.Parse` recusa segmento vazio, `.`, `..` e caracteres fora de `[A-Za-z0-9_.-]`. Mudança nessa validação precisa de teste cobrindo travessia de caminho.

## SEG-008 Mascaramento cobre toda saída
severity: error
stdout, stderr e mensagens de erro de execução passam por `mask.Masker` com todos os valores resolvidos naquela execução antes de voltar ao agente.

## SEG-009 Token só em memória
severity: error
O token desembrulhado não é gravado em disco, cache, arquivo de configuração nem variável de ambiente. Preferir `BAO_WRAPPING_TOKEN`, de uso único, a `BAO_TOKEN`.

## SEG-010 TLS e endereço do cofre
severity: error
Não desligar verificação TLS em código (`InsecureSkipVerify`, `BAO_SKIP_VERIFY` forçado). Configuração de TLS vem do ambiente, como no CLI `bao`.

## SEG-011 Saída e tempo limitados
Toda execução tem timeout e teto de saída (`Runner.Timeout`, `Runner.MaxOutput`). Remover ou elevar sem limite esses tetos abre caminho para travar o servidor MCP.

## SEG-012 Dependências novas justificadas
Dependência nova no `go.mod` precisa de motivo no PR. Preferir a biblioteca padrão e os SDKs oficiais (MCP Go SDK, cliente OpenBao).

## SEG-013 Repositório público: nada interno
severity: error
O repositório é público. Linha adicionada não cita empresa, domínio ou host interno, repositório privado nem endereço de cofre real. Exemplos usam `example.com`, `localhost` ou `127.0.0.1`.

## SEG-014 Exemplos e testes só com valores falsos
severity: error
Documentação, políticas de exemplo e testes usam valores obviamente falsos (`valor-de-teste`, `root-dev`). Nunca um token, chave ou URL de credencial real, nem parcial.
