#!/usr/bin/env bash
# Tutorial: temporary access. A wrapped token works once, a session ends when
# its token expires, without a token the tools fail predictably, and the
# token's expiry revokes the leases it created.
#   bash demo/tutorials/access/run.sh all | --check
source "$(dirname "$0")/../lib.sh"

export KEYHOLDER_SCOPE=my-project KEYHOLDER_ALLOW=curl
CASES=(01-single-use 02-expiry 03-no-token 04-lease-expiry)

setup() {
  echo_server
  admin kv put -mount=secret my-project/dev API_TOKEN=tok-fake-not-a-real-secret >/dev/null
  admin policy write my-project-read deploy/policy-project-read.hcl >/dev/null
  # A PKI mount whose certificates carry a lease: a secret with a lifetime
  # of its own, issued without any external service.
  admin secrets enable pki >/dev/null
  admin secrets tune -max-lease-ttl=2h pki >/dev/null
  admin write pki/root/generate/internal common_name=example.test ttl=2h >/dev/null 2>&1
  admin write pki/roles/agent allowed_domains=example.test allow_subdomains=true generate_lease=true max_ttl=1h >/dev/null 2>&1
  printf 'path "pki/issue/agent" { capabilities = ["update"] }\n' | admin policy write pki-issue - >/dev/null
}

# leases <prefix>: how many leases OpenBao holds under a path.
leases() {
  local keys
  keys="$(admin list -format=json "sys/leases/lookup/$1" 2>/dev/null)" || { echo 0; return; }
  printf '%s' "$keys" | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))'
}

case_01-single-use() {
  step 01-single-use "A wrapped token opens one session; reusing it fails"
  local w; w="$(wrapped_token my-project-read 1h)"
  show_as "BAO_WRAPPING_TOKEN=<wrapped token> khcall keyholder_status" env BAO_WRAPPING_TOKEN="$w" khcall keyholder_status
  show_as "BAO_WRAPPING_TOKEN=<same wrapped token> khcall keyholder_status" env BAO_WRAPPING_TOKEN="$w" khcall keyholder_status
}

case_02-expiry() {
  step 02-expiry "A 5-second token: the call works, then the same session is refused"
  session my-project-read 5s \
    keyholder_run '{"command":"curl","args":["-sS","-H","Authorization: Bearer {{vault://my-project/dev/API_TOKEN}}","http://127.0.0.1:18080/"]}' \
    +wait=7s \
    keyholder_status \
    keyholder_run '{"command":"curl","args":["-sS","-H","Authorization: Bearer {{vault://my-project/dev/API_TOKEN}}","http://127.0.0.1:18080/"]}'
}

case_03-no-token() {
  step 03-no-token "Without a token the server starts, explains, and refuses"
  show env -u BAO_TOKEN -u BAO_WRAPPING_TOKEN khcall keyholder_status keyholder_list
}

# issue_certificate <token>: a certificate with a 1-hour lease, issued with
# that token; prints the lease ID.
issue_certificate() {
  BAO_TOKEN="$1" bao write -field=lease_id pki/issue/agent common_name=agent.example.test ttl=1h
  echo
}

case_04-lease-expiry() {
  step 04-lease-expiry "A 5-second token creates a 1-hour lease; when the token expires, the lease goes with it"
  local t; t="$(admin token create -policy=pki-issue -ttl=5s -explicit-max-ttl=5s -field=token)"
  show_as "BAO_TOKEN=<5-second token> bao write -field=lease_id pki/issue/agent common_name=agent.example.test ttl=1h" \
    issue_certificate "$t"
  note "Leases under pki/issue/agent before the token expires: $(leases pki/issue/agent)"
  sleep 7
  show_as "BAO_TOKEN=<5-second token> bao token lookup" env BAO_TOKEN="$t" bao token lookup
  note "Leases under pki/issue/agent after the token expired: $(leases pki/issue/agent)"
}

main "$@"
