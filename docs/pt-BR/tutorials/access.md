# Tutorial: acesso temporário

## Objetivo

Ao final você vai ver que um token embrulhado funciona uma vez, que a sessão termina quando o token expira e que o servidor falha de forma previsível sem token.

Cada comando e cada saída abaixo vêm de uma execução real, gravada em `demo/tutorials/access/expected/` e conferida por `run.sh --check`. As saídas são do programa, em inglês.

## Como rodar

```bash
bash demo/tutorials/access/run.sh all
bash demo/tutorials/access/run.sh --check
```

Script: [`demo/tutorials/access/run.sh`](https://github.com/Mpaape/keyholder-mcp/blob/main/demo/tutorials/access/run.sh).

## 1. Um token embrulhado abre uma sessão

O primeiro servidor desembrulha o token. A segunda partida com o mesmo valor falha: o cofre recusa desembrulhar de novo. Um token embrulhado que vaza depois de usado não vale nada, e um token interceptado antes do uso aparece como esta falha.

```text
--8<-- "demo/tutorials/access/expected/01-single-use.txt"
```

## 2. Um token de 5 segundos expira no meio da sessão

Uma sessão: a primeira chamada funciona, `+wait=7s` pausa, `keyholder_status` informa a expiração e a chamada seguinte é recusada com uma mensagem que diz ao agente o que fazer.

```text
--8<-- "demo/tutorials/access/expected/02-expiry.txt"
```

## 3. Sem token

O servidor sobe mesmo assim, para o agente chamar `keyholder_status` e entender por que nada funciona. As outras tools recusam.

```text
--8<-- "demo/tutorials/access/expected/03-no-token.txt"
```
