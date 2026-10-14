# Segurança

O Keyholder MCP lida com credenciais. Se você encontrou uma falha que permite a um agente ver um valor de segredo, o token do cofre ou o token embrulhado, ou executar algo fora da lista de comandos permitidos, **não abra uma issue pública**.

Reporte pelo [aviso privado de vulnerabilidade do GitHub](https://github.com/Mpaape/keyholder-mcp/security/advisories/new), com os passos para reproduzir.

## Limites conhecidos (não são vulnerabilidades)

- O processo executado recebe o valor do segredo, como em qualquer uso real de credencial.
- O mascaramento só casa o valor exato; um comando que codifique o segredo o expõe. Por isso não se libera `sh`, `bash`, `python` ou `env` em `KEYHOLDER_ALLOW`.
- Não há proteção contra um agente hostil que roda com o mesmo usuário do sistema.

Detalhes no [modelo de ameaças](docs/SPEC.md#10-modelo-de-ameaças).
