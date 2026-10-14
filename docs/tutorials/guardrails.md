# Tutorial: guardrails

## Goal

At the end you will watch the project scope, the command allow list and the environment filter refuse what they should, even when the vault policy is broad.

Every command and output below comes from a real run, recorded in `demo/tutorials/guardrails/expected/` and checked by `run.sh --check`.

## Run it

```bash
bash demo/tutorials/guardrails/run.sh all
bash demo/tutorials/guardrails/run.sh --check
```

Script: [`demo/tutorials/guardrails/run.sh`](https://github.com/Mpaape/keyholder-mcp/blob/main/demo/tutorials/guardrails/run.sh).

## 1. Only the project scope

The token here has a broad policy on purpose (`secret/*`). `keyholder_list` still shows only `my-project`, and a reference to `other-project` is refused before any read.

```text
--8<-- "demo/tutorials/guardrails/expected/01-scope.txt"
```

## 2. The allow list

A command outside the list, a path instead of a name, and a placeholder as the command are all refused.

```text
--8<-- "demo/tutorials/guardrails/expected/02-allow-list.txt"
```

## 3. Reserved variables

The agent cannot set `BAO_*`, `VAULT_*` or `KEYHOLDER_*` for the child, for example to point it at another vault.

```text
--8<-- "demo/tutorials/guardrails/expected/03-reserved-env.txt"
```

## 4. The vault token never reaches the child

`printenv` is allowed in this tutorial only to prove the point; never allow it in real use. The server was started with `BAO_TOKEN` set, and the child does not have it.

```text
--8<-- "demo/tutorials/guardrails/expected/04-child-env.txt"
```
