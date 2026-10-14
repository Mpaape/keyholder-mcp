# Configuração

O `keyholder-mcp` não tem arquivo de configuração nem flags além de `-version`. Tudo vem de variáveis de ambiente definidas pelo cliente MCP que o inicia.

## Variáveis de ambiente

| Variável | Padrão | Significado |
| --- | --- | --- |
| `BAO_ADDR` | `https://127.0.0.1:8200` (padrão do cliente OpenBao) | Endereço do cofre |
| `BAO_WRAPPING_TOKEN` | — | Token embrulhado, de uso único. **Preferido.** Desembrulhado uma vez ao iniciar |
| `BAO_TOKEN` | — | Token comum. Aceito para testes locais; prefira o embrulhado |
| `KEYHOLDER_KV_MOUNT` | `secret` | Mount do KV **v2** que guarda os segredos |
| `KEYHOLDER_SCOPE` | vazio (tudo) | Prefixo de caminho do projeto. Referências fora dele são recusadas e não aparecem na listagem. Vazio gera um aviso no log |
| `KEYHOLDER_ALLOW` | vazio (nada) | Comandos que o agente pode rodar, separados por vírgula, pelo nome exato: `curl,aws,gcloud` |
| `KEYHOLDER_TIMEOUT` | `5m` | Tempo máximo por execução ([duração do Go](https://pkg.go.dev/time#ParseDuration): `30s`, `2m`) |

TLS e outras opções do cliente seguem as variáveis da CLI do OpenBao lidas pelo cliente oficial em Go (`BAO_CACERT`, `BAO_CLIENT_CERT`, `BAO_CLIENT_KEY`, `BAO_NAMESPACE`...). Esse cliente aceita os nomes `VAULT_*` como alternativa.

Ao iniciar, o servidor remove `BAO_WRAPPING_TOKEN`, `BAO_TOKEN` e `VAULT_TOKEN` do próprio ambiente, e todo processo filho recebe um ambiente sem `BAO_*`, `VAULT_*` e `KEYHOLDER_*`.

!!! danger "Nunca libere shells ou interpretadores"
    `KEYHOLDER_ALLOW` não pode ter `sh`, `bash`, `zsh`, `cmd`, `powershell`, `python`, `python3`, `node`, `env`, `printenv` nem nada que execute código recebido. Com eles o agente consegue imprimir ou codificar um segredo; veja [Mascaramento e seu limite](tutorials/masking.md).

## Referências

Uma referência aponta para um campo de um segredo no KV:

```text
vault://my-project/dev/API_TOKEN
        └─ caminho ──┘ └─chave─┘
```

- **caminho**: o caminho do segredo dentro do mount (`secret/my-project/dev` → `my-project/dev`).
- **chave**: o campo dentro do segredo.
- Os segmentos usam `A-Z a-z 0-9 _ . -`; segmento vazio, `.` e `..` são recusados.
- O valor precisa ser texto.

No `keyholder_run`, a referência vai dentro de um placeholder, `{{vault://caminho/CHAVE}}`, em qualquer ponto de um argumento ou do valor de uma variável. Cabem vários placeholders numa mesma string.

## Tools

### `keyholder_status`

Sem entrada. Saída:

| Campo | Significado |
| --- | --- |
| `addr` | Endereço do cofre |
| `initialized`, `sealed`, `server_version` | De `sys/health` |
| `kv_mount`, `scope` | A configuração em uso |
| `token_ttl_seconds`, `token_expire_time` | Tempo de vida restante do token da sessão |
| `token_renewable`, `token_policies` | Da consulta ao token |
| `session_error` | Por que não há sessão utilizável: sem token, desembrulho recusado, expirado ou sem permissão para consultar a si mesmo |

Nunca devolve o token nem o accessor.

### `keyholder_list`

Sem entrada. Saída: `scope`, `entries` (cada uma com `ref` e `version`) e um `note` para o agente. Percorre o escopo (até 500 entradas), lendo cada segredo para saber os nomes dos campos e descartando os valores.

### `keyholder_run`

Entrada:

| Campo | Tipo | Significado |
| --- | --- | --- |
| `command` | texto | Nome de um comando permitido. Sem caminho, sem placeholder |
| `args` | lista de textos | Argumentos; podem conter `{{vault://...}}` |
| `env` | mapa | Variáveis extras para o filho; os valores podem conter `{{vault://...}}`. Nomes `BAO_*`, `VAULT_*`, `KEYHOLDER_*` são recusados |

Saída:

| Campo | Significado |
| --- | --- |
| `exit_code` | Código de saída do filho |
| `stdout`, `stderr` | Até 64 KiB cada, com todo valor resolvido trocado por `[hidden:<referência>]` |
| `refs_used` | Referências resolvidas nesta execução |
| `truncated` | A saída passou de 64 KiB |
| `timed_out` | A execução atingiu `KEYHOLDER_TIMEOUT` |

O comando roda **sem shell**: `exec` com a lista de argumentos e stdin no dispositivo nulo. Pipes, redirecionamentos e `$VAR` não são interpretados, e caracteres especiais dentro de um segredo nunca viram código.

Erros (não permitido, fora do escopo, referência inexistente, acesso negado) voltam como erro da tool, com uma mensagem que o agente consegue usar, sem URLs nem corpo de resposta do cofre.

## Política

A [política de exemplo](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-project-read.hcl) concede exatamente o que as três tools precisam para um projeto:

| Caminho | Permissões | Usado por |
| --- | --- | --- |
| `secret/data/my-project/*` | read | `keyholder_run` (resolver), `keyholder_list` (nomes dos campos) |
| `secret/metadata/my-project`, `.../*` | list, read | `keyholder_list` |
| `auth/token/lookup-self` | read | `keyholder_status` (já está na política `default` do OpenBao) |

Sem `lookup-self`, o `keyholder_status` estima o TTL pela resposta do desembrulho e avisa isso em `session_error`.

## Trilha de auditoria

Cada `keyholder_run` escreve uma linha no stderr do servidor, que os clientes MCP guardam nos logs do servidor:

```text
2026/10/13 22:35:40 audit cmd="curl" refs=[vault://my-project/dev/API_TOKEN] exit=0 dur=8ms status=ok
```

Ela registra o nome do comando, as referências, o código de saída, a duração e o resultado. Nunca valores, nunca argumentos. Os [audit devices](https://openbao.org/docs/audit/) do próprio OpenBao registram cada leitura do lado do cofre.

## khcall

O `khcall` é um cliente MCP mínimo para depuração e para os tutoriais:

```bash
khcall [-server keyholder-mcp] -tools
khcall [-server keyholder-mcp] <tool> ['<argumentos json>'] [+wait=<duração>] [<tool> ...]
```

Todas as chamadas rodam numa sessão (um servidor, um desembrulho). `+wait=7s` pausa entre chamadas. Código de saída: 0 quando todas dão certo, 1 quando uma tool reporta erro, 2 em erro de uso ou de protocolo.
