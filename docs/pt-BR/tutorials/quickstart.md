# Tutorial: início rápido

## Objetivo

Ao final você vai guardar um segredo, emitir um token embrulhado e ver um agente listar a credencial e chamar uma API com ela, sem que o valor volte.

Cada comando e cada saída abaixo vêm de uma execução real, gravada em `demo/tutorials/quickstart/expected/` e conferida por `run.sh --check`. As saídas são do programa, em inglês.

## Como rodar

```bash
bash demo/tutorials/quickstart/run.sh all
bash demo/tutorials/quickstart/run.sh --check
```

Script: [`demo/tutorials/quickstart/run.sh`](https://github.com/Mpaape/keyholder-mcp/blob/main/demo/tutorials/quickstart/run.sh).

## 1. O humano guarda um segredo e uma política

Fora do chat, como administrador do cofre. `API_URL` também é guardada: qualquer valor pode virar referência, não só senhas.

```text
--8<-- "demo/tutorials/quickstart/expected/01-store.txt"
```

## 2. O agente vê as tools, a sessão e os nomes

Cada `khcall` é uma sessão MCP, e cada sessão usa um token embrulhado novo. `keyholder_status` informa o TTL e as políticas, mas nunca o token; `keyholder_list` devolve só nomes.

```text
--8<-- "demo/tutorials/quickstart/expected/02-session.txt"
```

## 3. O agente chama uma API com o token

A API de teste responde com o cabeçalho que recebeu. O agente lê `[hidden:...]`; a última linha confirma, por hash, que a API recebeu o valor real guardado. A linha de auditoria no stderr nomeia o comando e as referências, nunca o valor.

```text
--8<-- "demo/tutorials/quickstart/expected/03-run.txt"
```
