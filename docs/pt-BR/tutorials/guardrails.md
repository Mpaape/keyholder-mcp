# Tutorial: proteções

## Objetivo

Ao final você vai ver o escopo do projeto, a lista de comandos permitidos e o filtro de ambiente recusarem o que devem, mesmo com uma política ampla no cofre.

Cada comando e cada saída abaixo vêm de uma execução real, gravada em `demo/tutorials/guardrails/expected/` e conferida por `run.sh --check`. As saídas são do programa, em inglês.

## Como rodar

```bash
bash demo/tutorials/guardrails/run.sh all
bash demo/tutorials/guardrails/run.sh --check
```

Script: [`demo/tutorials/guardrails/run.sh`](https://github.com/Mpaape/keyholder-mcp/blob/main/demo/tutorials/guardrails/run.sh).

## 1. Só o escopo do projeto

O token aqui tem uma política ampla de propósito (`secret/*`). Mesmo assim `keyholder_list` mostra só `my-project`, e uma referência a `other-project` é recusada antes de qualquer leitura.

```text
--8<-- "demo/tutorials/guardrails/expected/01-scope.txt"
```

## 2. A lista de comandos permitidos

Comando fora da lista, caminho no lugar do nome e placeholder como comando: todos recusados.

```text
--8<-- "demo/tutorials/guardrails/expected/02-allow-list.txt"
```

## 3. Variáveis reservadas

O agente não consegue definir `BAO_*`, `VAULT_*` ou `KEYHOLDER_*` para o processo filho, por exemplo para apontá-lo para outro cofre.

```text
--8<-- "demo/tutorials/guardrails/expected/03-reserved-env.txt"
```

## 4. O token do cofre nunca chega ao processo filho

`printenv` está liberado neste tutorial só para provar o ponto; nunca o libere em uso real. O servidor subiu com `BAO_TOKEN` definido, e o filho não o recebe.

```text
--8<-- "demo/tutorials/guardrails/expected/04-child-env.txt"
```
