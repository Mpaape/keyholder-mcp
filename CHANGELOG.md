# Changelog

## Unreleased

- CI com `gofmt`, `go vet`, `go test` e `go build` em Linux, Windows e macOS.
- Servidor MCP em Go com as tools `keyholder_status`, `keyholder_listar` e `keyholder_executar`.
- Runner com placeholders `{{vault://...}}` resolvidos só na execução, sem shell, com saída mascarada e limites de tempo e de saída.
- Token embrulhado de uso único via `BAO_WRAPPING_TOKEN`; `BAO_*`, `VAULT_*` e `KEYHOLDER_*` fora do processo filho.
- Política de exemplo para o OpenBao em `deploy/`.
- Revisão de PR com AurumCode: regras de segurança, review e SOLID; gitleaks e Semgrep.
- Licença MIT.
