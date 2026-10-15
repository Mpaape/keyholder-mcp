# Tutoriais

Todo tutorial é **executável**. Os comandos estão em `demo/tutorials/<tutorial>/run.sh`, cada caso grava a saída real da última execução em `out/`, e `run.sh --check` compara `out/` com `expected/`. As saídas destas páginas são incluídas direto de `expected/` na hora do build do site, então não ficam desatualizadas. O CI roda todos os tutoriais e a conferência a cada mudança.

Valores que mudam entre execuções aparecem pela forma: `<timestamp>`, `<duration>`, `<ttl>`, `<time>`, e `<wrapped token>` para o token, que nunca é impresso.

| Tutorial | Casos |
| --- | --- |
| [Início rápido](quickstart.md) | 1. o humano guarda um segredo e uma política; 2. o agente vê as tools, a sessão e os nomes; 3. o agente chama uma API com o token, mascarado |
| [Acesso temporário](access.md) | 1. um token embrulhado abre uma sessão, reusar falha; 2. um token de 5 segundos expira no meio da sessão; 3. sem token o servidor explica e recusa; 4. a expiração revoga os leases que o token criou |
| [Proteções](guardrails.md) | 1. só o escopo do projeto, mesmo com política ampla; 2. comando fora da lista, com caminho ou com placeholder; 3. variáveis reservadas; 4. o token do cofre nunca chega ao processo filho |
| [Mascaramento e seu limite](masking.md) | 1. um valor refletido é mascarado; 2. LIMITE: um interpretador liberado codifica o valor e escapa do mascaramento |
| [Credenciais de nuvem](cloud.md) | 1. o humano habilita o engine da AWS; 2. o agente vê o caminho dinâmico; 3. a AWS CLI roda com uma credencial criada para a execução e revogada depois; 4. caminhos fora da lista são recusados; 5. o gcloud recebe o token por um arquivo temporário |

## Como rodar

Precisa de `bash`, `go`, `bao` (CLI do OpenBao), `curl` e `python3`; o tutorial de nuvem também precisa de `aws`, `gcloud`, [moto](https://github.com/getmoto/moto) e do plugin de AWS do OpenBao (veja a página dele). Nada toca um cofre real: cada execução sobe o próprio OpenBao de desenvolvimento, em memória, em `127.0.0.1:18200`, com segredos falsos, e o derruba ao final.

```bash
bash demo/tutorials/quickstart/run.sh all       # todos os casos; grava out/
bash demo/tutorials/quickstart/run.sh --check   # compara out/ com expected/
bash demo/tutorials/quickstart/run.sh 03-run    # um caso só
```

`KH_BIN_DIR=<dir>` usa `keyholder-mcp` e `khcall` já compilados em vez de compilar.
