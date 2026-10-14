O Keyholder MCP deixa agentes de IA usarem credenciais do OpenBao sem ver os valores.
O agente só conhece referências `vault://caminho/CHAVE`; o valor é resolvido no runner,
na hora de executar um comando permitido, e a saída volta mascarada.

Ao revisar, a pergunta principal é: esta mudança pode fazer um valor de segredo, o token
do cofre ou o token embrulhado chegar ao agente, a um log, a um erro, a um arquivo ou a um
processo que não deveria recebê-lo? Siga o caminho do dado desde `bao.Store.Resolve` até
a resposta MCP antes de concluir.

Pontos sensíveis do código:
- `internal/runner`: lista de comandos permitidos, execução sem shell, filtro de ambiente
  (`BAO_*`, `VAULT_*`, `KEYHOLDER_*`), limite de saída e mascaramento.
- `internal/bao`: desembrulho do token, remoção das variáveis de token do ambiente,
  escopo das referências e mensagens de erro sem URL nem corpo da resposta.
- `cmd/keyholder-mcp`: o que cada tool devolve e o que vai para o log de auditoria.
- `internal/ref`: validação de referências (sem `..`, sem segmento vazio).

Limites aceitos pelo projeto: o processo executado recebe o valor; o mascaramento só casa
o valor exato; não há proteção contra um agente hostil com o mesmo usuário do sistema.
Não aponte esses limites como defeito, a menos que a mudança os piore.

Comentários e mensagens de erro do projeto são em português.
Use os testes existentes para sustentar cada achado e indique quando faltar teste.
