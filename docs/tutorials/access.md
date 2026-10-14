# Tutorial: temporary access

## Goal

At the end you will see that a wrapped token works once, that a session ends when its token expires, and that the server fails predictably without a token.

Every command and output below comes from a real run, recorded in `demo/tutorials/access/expected/` and checked by `run.sh --check`.

## Run it

```bash
bash demo/tutorials/access/run.sh all
bash demo/tutorials/access/run.sh --check
```

Script: [`demo/tutorials/access/run.sh`](https://github.com/Mpaape/keyholder-mcp/blob/main/demo/tutorials/access/run.sh).

## 1. A wrapped token opens one session

The first server unwraps the token. The second start with the same value fails: the vault refuses to unwrap it again. A wrapped token that leaks after use is worthless, and a token that was intercepted before use shows up as this failure.

```text
--8<-- "demo/tutorials/access/expected/01-single-use.txt"
```

## 2. A 5-second token expires mid-session

One session: the first call works, `+wait=7s` pauses, then `keyholder_status` reports the expiry and the next call is refused with a message that tells the agent what to do.

```text
--8<-- "demo/tutorials/access/expected/02-expiry.txt"
```

## 3. No token

The server still starts, so the agent can call `keyholder_status` and learn why nothing works. The other tools refuse.

```text
--8<-- "demo/tutorials/access/expected/03-no-token.txt"
```
