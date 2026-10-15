# Persistent vault

The dev server from [Getting started](getting-started.md) keeps everything in memory and has no TLS. For real secrets, run OpenBao as a persistent server. This page sets up one node on your machine, the setup Keyholder is built for: one person, one vault, the agent on the same machine.

Every command here also runs in the [persistent vault tutorial](tutorials/server.md), against the same [`deploy/server.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/server.hcl), in CI. The outputs are there.

## What this setup decides

| Question | Choice | Why |
| --- | --- | --- |
| Storage | Integrated storage (Raft), one node | Data on disk, survives restarts, one-command backups; no other service to run |
| Network | TLS on `127.0.0.1:8200` only | Only this machine reaches the vault; the agent's commands run here too |
| Unsealing | You, after every restart, with 2 of 3 keys (Shamir) | No cloud account, no key on the same disk as the data; the vault is sealed whenever the machine was off |
| Root token | Used once, then revoked | An administrator login with a narrow policy does the daily work |

**Auto unseal** (a cloud KMS, an HSM, or another OpenBao's transit engine) is supported by OpenBao and changes only a `seal` block in the config: see [seal concepts](https://openbao.org/docs/concepts/seal/). Keyholder does not test it; it makes sense on a shared server, which is not this setup.

On Windows, run the server inside WSL or a container: see [Windows and WSL](windows-wsl.md).

## 1. Files and certificate

```bash
sudo mkdir -p /etc/openbao/tls /var/lib/openbao/data
sudo cp deploy/server.hcl /etc/openbao/server.hcl
```

The certificate must name `127.0.0.1`. With a company certificate authority, ask it for one. Otherwise make a local authority and keep its key off the machine afterwards:

```bash
cd /etc/openbao/tls
sudo openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -days 3650 \
  -subj /CN=keyholder-local-ca -keyout ca.key -out ca.pem
sudo openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -subj /CN=127.0.0.1 \
  -keyout tls.key -out tls.csr
printf 'subjectAltName=IP:127.0.0.1,DNS:localhost\n' | sudo tee ext >/dev/null
sudo openssl x509 -req -in tls.csr -CA ca.pem -CAkey ca.key -CAcreateserial -days 825 \
  -extfile ext -out tls.pem
```

Every client of the vault, the `bao` CLI and Keyholder alike, trusts it through two variables:

```bash
export BAO_ADDR=https://127.0.0.1:8200 BAO_CACERT=/etc/openbao/tls/ca.pem
```

## 2. Start the server

```bash
bao server -config=/etc/openbao/server.hcl
```

To start it with the machine, a systemd unit is enough. This one is an example; the tutorial starts the server directly:

```ini
# /etc/systemd/system/openbao.service
[Unit]
Description=OpenBao for Keyholder
After=network.target

[Service]
ExecStart=/usr/local/bin/bao server -config=/etc/openbao/server.hcl
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

The server always starts **sealed**. Until it is unsealed it holds no keys in memory and answers nothing but its status.

## 3. Initialize once

```bash
bao operator init -key-shares=3 -key-threshold=2
```

This prints 3 unseal keys and the root token, once. Any 2 keys unseal the vault; 1 alone cannot. Put the keys in different places: a password manager, a printed copy in a drawer, a second person. If all 3 are lost, the data cannot be recovered.

Then unseal, entering two different keys:

```bash
bao operator unseal     # asks for a key
bao operator unseal     # a different key
```

## 4. Set up and revoke the root token

With the root token, once:

```bash
export BAO_TOKEN=<root token>
bao secrets enable -path=secret kv-v2
bao policy write my-project-read deploy/policy-project-read.hcl
bao policy write keyholder-admin deploy/policy-admin.hcl
bao auth enable userpass
bao write auth/userpass/users/admin password=<password> token_policies=keyholder-admin token_ttl=8h token_max_ttl=8h
bao token revoke -self
unset BAO_TOKEN
```

[`policy-admin.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-admin.hcl) lets the administrator store secrets, write policies, issue tokens and take backups, and nothing else. If you ever need the root token again, generate a new one with the unseal keys (`bao operator generate-root`).

## 5. Every day

Log in, store what the agent will use, and issue its wrapped token:

```bash
bao login -method=userpass username=admin
bao kv put -mount=secret my-project/dev API_TOKEN=...
export BAO_WRAPPING_TOKEN="$(bao token create -policy=my-project-read -ttl=8h -explicit-max-ttl=8h -wrap-ttl=5m -field=wrapping_token)"
```

Then start your MCP client with `BAO_ADDR`, `BAO_CACERT`, `KEYHOLDER_SCOPE` and `KEYHOLDER_ALLOW` set, as in [MCP clients](clients.md).

## 6. After a restart

The vault comes back sealed. Keyholder says so in `keyholder_status`, with `"sealed": true` and a session error that asks for the unseal, and it does not consume the wrapped token. Unseal with 2 keys, then restart the MCP server (or reconnect it in the client) so it opens the session.

## 7. Backups

```bash
bao operator raft snapshot save keyholder-$(date +%F).snap
```

The snapshot is encrypted with the vault's keys. Restore it with `bao operator raft snapshot restore` and unseal with the same unseal keys. Keep snapshots and unseal keys apart.

## What this setup does not cover

- **A second factor** for the administrator login. OpenBao supports TOTP for logins; Keyholder has not chosen how to use it yet ([decision 2](spec.md#9-open-decisions)).
- **A shared vault** for a team, with high availability and auto unseal. See the [OpenBao documentation](https://openbao.org/docs/).
