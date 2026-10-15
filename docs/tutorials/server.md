# Tutorial: persistent vault

## Goal

At the end you will see an OpenBao server that keeps its data on disk and speaks TLS: initialized once, unsealed with 2 of 3 keys, set up without keeping the root token, used by Keyholder over TLS, sealed again by a restart, unsealed, and backed up.

Every command and output below comes from a real run, recorded in `demo/tutorials/server/expected/` and checked by `run.sh --check`. The steps, and why each one is there, are in the [persistent vault guide](../server.md).

**What is different from a real install.** The server runs [`deploy/server.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/server.hcl) itself, with its paths moved to a temporary directory and its port moved to 18200. The certificate authority is created for the run. The unseal keys, the root token and the administrator's password stay in that temporary directory and are deleted with it.

## Run it

```bash
bash demo/tutorials/server/run.sh all       # needs bao, openssl, python3, curl and go
bash demo/tutorials/server/run.sh --check
```

Script: [`demo/tutorials/server/run.sh`](https://github.com/Mpaape/keyholder-mcp/blob/main/demo/tutorials/server/run.sh). Policies: [`policy-project-read.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-project-read.hcl) and [`policy-admin.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-admin.hcl).

## 1. Initialize once, unseal with 2 of 3 keys

A new server is sealed and not initialized. `bao operator init` runs once in the vault's life and prints the unseal keys and the root token; here they go to a file that never enters the repository. Each unseal key moves the progress by one; the second one opens the vault.

```text
--8<-- "demo/tutorials/server/expected/01-init.txt"
```

## 2. Set up, then revoke the root token

The root token creates the KV mount, the policies and an administrator login, and is then revoked. From then on the administrator logs in with a password and gets an 8-hour token that can store secrets and issue the agents' wrapped tokens, but cannot change the vault's configuration.

```text
--8<-- "demo/tutorials/server/expected/02-setup.txt"
```

## 3. Keyholder over TLS

With `BAO_CACERT` pointing at the certificate authority, the session opens and the API receives the token. Without it, Keyholder refuses the connection and says which variable to set.

```text
--8<-- "demo/tutorials/server/expected/03-tls.txt"
```

## 4. A restart seals the vault

A wrapped token is issued, then the server restarts. It comes back sealed: Keyholder reports it and does not consume the token. After two unseal keys a new Keyholder start opens the session with that same token, and the secret stored before the restart is still there.

```text
--8<-- "demo/tutorials/server/expected/04-restart.txt"
```

## 5. Back up

The administrator saves a snapshot of the whole vault. It is encrypted with the vault's keys, so it is restored and unsealed with the same unseal keys.

```text
--8<-- "demo/tutorials/server/expected/05-backup.txt"
```
