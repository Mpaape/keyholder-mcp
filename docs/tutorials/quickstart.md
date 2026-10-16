# Tutorial: quickstart

## Goal

At the end you will store a secret, issue a wrapped token, and watch an agent list the credential and call an API with it, without the value ever coming back.

Every command and output below comes from a real run, recorded in `demo/tutorials/quickstart/expected/` and checked by `run.sh --check`.

## Run it

```bash
bash demo/tutorials/quickstart/run.sh all
bash demo/tutorials/quickstart/run.sh --check
```

Script: [`demo/tutorials/quickstart/run.sh`](https://github.com/Mpaape/keyholder-mcp/blob/main/demo/tutorials/quickstart/run.sh).

## 1. The human stores a secret and a policy

Outside the chat, as the vault administrator. `API_URL` is stored too: any value can be a reference, not only passwords.

```text
--8<-- "demo/tutorials/quickstart/expected/01-store.txt"
```

## 2. The agent sees the tools, the session and the names

Each `khcall` is one MCP session, and each session uses a fresh wrapped token. `keyholder_status` reports the TTL and policies but never the token; `keyholder_list` returns names only.

```text
--8<-- "demo/tutorials/quickstart/expected/02-session.txt"
```

## 3. The agent calls an API with the token

The test API answers with the header it received. The agent reads `[hidden:...]`; the last line confirms, by hash, that the API received the real stored value. The audit line on stderr names the command and the references, never the value.

```text
--8<-- "demo/tutorials/quickstart/expected/03-run.txt"
```

## 4. A credential file as an argument

`curl -K` reads its options from a file. The file is declared in `files`, holds the header with the token, exists only during the run, and `{{file:CFG}}` is its path. The API still receives the real token; the agent reads it masked.

```text
--8<-- "demo/tutorials/quickstart/expected/04-file.txt"
```
