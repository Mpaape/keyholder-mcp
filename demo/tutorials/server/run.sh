#!/usr/bin/env bash
# Tutorial: persistent vault. OpenBao from deploy/server.hcl, with data on
# disk and TLS: initialize, unseal, set up without the root token, use it
# from Keyholder over TLS, restart (sealed again), unseal, back up.
#   bash demo/tutorials/server/run.sh all | --check
source "$(dirname "$0")/../lib.sh"

export KEYHOLDER_SCOPE=my-project KEYHOLDER_ALLOW=curl
export BAO_ADDR="https://127.0.0.1:18200" BAO_CACERT="$WORK/tls/ca.pem"
CASES=(01-init 02-setup 03-tls 04-restart 05-backup)
SERVER_PID=""

# The dev server is not used here: each case drives the real server.
start_vault() { :; }

setup() {
  echo_server
  # A local certificate authority and a certificate for 127.0.0.1. With a
  # company CA, only BAO_CACERT changes.
  mkdir -p "$WORK/tls" "$WORK/data"
  openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -days 2 \
    -subj /CN=keyholder-tutorial-ca -keyout "$WORK/tls/ca.key" -out "$WORK/tls/ca.pem" 2>/dev/null
  openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -subj /CN=127.0.0.1 \
    -keyout "$WORK/tls/tls.key" -out "$WORK/tls/tls.csr" 2>/dev/null
  printf 'subjectAltName=IP:127.0.0.1,DNS:localhost\n' >"$WORK/tls/ext"
  openssl x509 -req -in "$WORK/tls/tls.csr" -CA "$WORK/tls/ca.pem" -CAkey "$WORK/tls/ca.key" \
    -CAcreateserial -days 2 -extfile "$WORK/tls/ext" -out "$WORK/tls/tls.pem" 2>/dev/null
  # deploy/server.hcl itself, with the paths and ports moved into $WORK.
  sed -e "s#/var/lib/openbao/data#$WORK/data#" -e "s#/etc/openbao/tls#$WORK/tls#" \
      -e 's#127.0.0.1:8200#127.0.0.1:18200#g' -e 's#127.0.0.1:8201#127.0.0.1:18201#' \
      deploy/server.hcl >"$WORK/server.hcl"
}

# server_start / server_stop run `bao server -config` like a service would.
server_start() {
  bao server -config="$WORK/server.hcl" >>"$WORK/bao.log" 2>&1 &
  SERVER_PID=$!; PIDS+=("$SERVER_PID")
  local rc
  for _ in $(seq 1 50); do
    rc=0; bao status >/dev/null 2>&1 || rc=$?
    [ "$rc" -ne 1 ] && return 0   # 0 unsealed, 2 sealed; 1 is not answering yet
    sleep 0.2
  done
  echo "OpenBao did not start"; cat "$WORK/bao.log"; return 1
}
server_stop() { kill "$SERVER_PID"; wait "$SERVER_PID" 2>/dev/null || true; }

# state: initialized and sealed, read from `bao status -format=json`.
state() {
  bao status -format=json 2>/dev/null | python3 -c '
import json, sys
s = json.load(sys.stdin)
print("initialized=%s sealed=%s unseal_progress=%s/%s" % (str(s["initialized"]).lower(), str(s["sealed"]).lower(), s["progress"], s["t"]))'
}

# unseal_key <n>: the n-th unseal key from the init output (kept in $WORK).
unseal_key() { python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["unseal_keys_b64"][int(sys.argv[2])-1])' "$WORK/init.json" "$1"; }

# The commands below print keys or tokens; their output goes to a file in
# $WORK, never to the records.
unseal_quietly() { bao operator unseal "$1" >/dev/null; }
init_to() { bao operator init -key-shares=3 -key-threshold=2 -format=json >"$1"; }
admin_login() { HOME="$WORK" bao login -method=userpass -field=token username=admin password="$1" >"$2"; }

unseal() {
  local n
  for n in "$@"; do
    show_as "bao operator unseal <unseal key $n> >/dev/null" unseal_quietly "$(unseal_key "$n")"
    note "Vault: $(state)"
  done
  # Wait until the node is the active one and answers requests.
  for _ in $(seq 1 50); do bao read -format=json sys/leader 2>/dev/null | grep -q '"is_self": true' && return 0; sleep 0.2; done
}

case_01-init() {
  step 01-init "Start the server, initialize it once, unseal it with 2 of 3 keys"
  server_start
  show bao status
  show_as "bao operator init -key-shares=3 -key-threshold=2 -format=json > <file kept offline>" \
    init_to "$WORK/init.json"
  note "The file now holds 3 unseal keys and the root token. It never goes into a repository; give each key to a different place or person."
  unseal 1 2
}

case_02-setup() {
  step 02-setup "With the root token: KV, policies and an administrator login; then the root token is revoked"
  ROOT_TOKEN="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["root_token"])' "$WORK/init.json")"
  show admin secrets enable -path=secret kv-v2
  show admin kv put -mount=secret my-project/dev API_TOKEN=tok-fake-not-a-real-secret
  show admin policy write my-project-read deploy/policy-project-read.hcl
  show admin policy write keyholder-admin deploy/policy-admin.hcl
  show admin auth enable userpass
  openssl rand -base64 24 >"$WORK/admin.password"
  show_as "bao write auth/userpass/users/admin password=<password> token_policies=keyholder-admin token_ttl=8h token_max_ttl=8h" \
    admin write auth/userpass/users/admin password="$(cat "$WORK/admin.password")" token_policies=keyholder-admin token_ttl=8h token_max_ttl=8h
  show admin token revoke -self
  note "The root token no longer exists. Each day the administrator logs in instead:"
  show_as "bao login -method=userpass -field=token username=admin password=<password> > <administrator token>" \
    admin_login "$(cat "$WORK/admin.password")" "$WORK/admin.token"
  ROOT_TOKEN="$(cat "$WORK/admin.token")"   # from here on, admin is the administrator
  note "Policies of the administrator's token: $(admin token lookup -format=json | python3 -c 'import json,sys; print(", ".join(json.load(sys.stdin)["data"]["policies"]))')"
}

case_03-tls() {
  step 03-tls "Keyholder reaches the vault over TLS; without the CA it refuses to connect"
  session my-project-read 1h keyholder_status \
    keyholder_run '{"command":"curl","args":["-sS","-H","Authorization: Bearer {{vault://my-project/dev/API_TOKEN}}","http://127.0.0.1:18080/"]}'
  note "Did the API receive the stored token? $(server_saw 'Bearer tok-fake-not-a-real-secret')"
  local w; w="$(wrapped_token my-project-read 1h)"
  show_as "BAO_CACERT= BAO_WRAPPING_TOKEN=<wrapped token> khcall keyholder_status" \
    env -u BAO_CACERT BAO_WRAPPING_TOKEN="$w" khcall keyholder_status
}

case_04-restart() {
  step 04-restart "After a restart the vault is sealed: Keyholder fails cleanly, then works once it is unsealed"
  local w; w="$(wrapped_token my-project-read 1h)"
  note "A wrapped token was issued before the restart."
  server_stop
  server_start
  note "Vault after the restart: $(state)"
  show_as "BAO_WRAPPING_TOKEN=<same wrapped token> khcall keyholder_status" env BAO_WRAPPING_TOKEN="$w" khcall keyholder_status
  unseal 2 3
  show_as "BAO_WRAPPING_TOKEN=<same wrapped token> khcall keyholder_list" env BAO_WRAPPING_TOKEN="$w" khcall keyholder_list
}

case_05-backup() {
  step 05-backup "A snapshot of the whole vault, taken by the administrator"
  show_as "bao operator raft snapshot save <backup file>" admin operator raft snapshot save "$WORK/backup.snap"
  note "Snapshot written: $([ -s "$WORK/backup.snap" ] && echo yes || echo no). It is encrypted with the vault's keys; it is restored with \`bao operator raft snapshot restore\` and unsealed with the same keys."
}

main "$@"
