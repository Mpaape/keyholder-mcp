# Tutorial: masking and its limit

## Goal

At the end you will see masking work on a value the API reflects back, and see exactly where it stops working.

Every command and output below comes from a real run, recorded in `demo/tutorials/masking/expected/` and checked by `run.sh --check`.

## Run it

```bash
bash demo/tutorials/masking/run.sh all
bash demo/tutorials/masking/run.sh --check
```

Script: [`demo/tutorials/masking/run.sh`](https://github.com/Mpaape/keyholder-mcp/blob/main/demo/tutorials/masking/run.sh).

## 1. A reflected value is masked

The API puts the received header in its answer. The agent reads the reference, not the value.

```text
--8<-- "demo/tutorials/masking/expected/01-reflected.txt"
```

## 2. LIMIT: encoding escapes the mask

`python3` is allowed here **only** to show the limit. The interpreter base64-encodes the value and the agent reads it, unmasked. Masking only matches the exact value; this is why the allow list must never include interpreters or shells (`sh`, `bash`, `python3`, `node`, `env`).

```text
--8<-- "demo/tutorials/masking/expected/02-limit-encoded.txt"
```
