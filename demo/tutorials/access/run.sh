#!/usr/bin/env bash
# Tutorial: temporary access. A wrapped token works once, a session ends when
# its token expires, and without a token the tools fail predictably.
#   bash demo/tutorials/access/run.sh all | --check
source "$(dirname "$0")/../lib.sh"

export KEYHOLDER_SCOPE=my-project KEYHOLDER_ALLOW=curl
CASES=(01-single-use 02-expiry 03-no-token)

setup() {
  echo_server
  admin kv put -mount=secret my-project/dev API_TOKEN=tok-fake-0123456789 >/dev/null
  admin policy write my-project-read deploy/policy-project-read.hcl >/dev/null
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

main "$@"
