---
name: review
version: 1
paths: ["cmd/**", "internal/**"]
---
Regras gerais de revisão de código Go do Keyholder MCP.

## REV-001 Erros tratados e com contexto
severity: error
Nenhum erro ignorado com `_` sem comentário que justifique. Erros são embrulhados com `fmt.Errorf("contexto: %w", err)` para permitir `errors.Is`/`errors.As`, sem expor segredos (ver seguranca#seg-002).

## REV-002 Comportamento novo vem com teste
severity: error
Mudança de comportamento em `internal/` traz teste que falharia sem ela. Bug corrigido traz teste de regressão. Testes não dependem de rede externa nem de shell; o runner usa o próprio binário de teste como processo filho.

## REV-003 Contexto propagado
Funções que fazem I/O recebem `context.Context` como primeiro parâmetro e o repassam (`ReadWithContext`, `CommandContext`). Não criar `context.Background()` no meio de uma chamada que já tem contexto.

## REV-004 stdout é do protocolo MCP
severity: error
O servidor fala MCP por stdio: nada pode escrever em `os.Stdout` além do SDK. Logs e auditoria vão para `os.Stderr` via `log`.

## REV-005 Sem panic em caminho de requisição
severity: error
Handlers de tool e o runner devolvem erro em vez de chamar `panic`, `log.Fatal` ou `os.Exit`. `log.Fatal` só na inicialização, em `main`.

## REV-006 Portabilidade Windows, Linux e macOS
Nada de caminho com `/` fixo, `sh`, sinais POSIX ou comandos do sistema em código e testes. Usar `filepath`, `os.PathListSeparator` e `exec.LookPath`.

## REV-007 Mensagens ao agente são acionáveis
Erros devolvidos às tools dizem o que aconteceu e o próximo passo (por exemplo, renovar o acesso fora do chat), em português, sem URL interna nem corpo de resposta HTTP.

## REV-008 Código formatado e verificado
Código passa em `gofmt` e `go vet`. Nomes exportados têm comentário de documentação.

## REV-009 Spec e README acompanham a mudança
Mudança em tool MCP, variável de ambiente ou fluxo de acesso atualiza `README.md` e `docs/SPEC.md` no mesmo PR.
