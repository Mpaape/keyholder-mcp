#!/usr/bin/env bash
# Tutorial: quickstart. Store a secret, issue a wrapped token, let an agent
# list and use it, and check that the value never comes back.
#   bash demo/tutorials/quickstart/run.sh all      # runs every case, writes out/
#   bash demo/tutorials/quickstart/run.sh --check  # compares out/ with expected/
source "$(dirname "$0")/../lib.sh"

export KEYHOLDER_SCOPE=my-project KEYHOLDER_ALLOW=curl
CASES=(01-store 02-session 03-run)

setup() { echo_server; }

case_01-store() {
  step 01-store "The human stores a secret and a read policy (outside the chat)"
  show admin kv put -mount=secret my-project/dev API_TOKEN=tok-fake-not-a-real-secret API_URL=http://127.0.0.1:18080/health
  show admin policy write my-project-read deploy/policy-project-read.hcl
}

case_02-session() {
  step 02-session "The agent sees the tools, the session and the reference names"
  session my-project-read 1h -tools
  session my-project-read 1h keyholder_status keyholder_list
}

case_03-run() {
  step 03-run "The agent calls an API with the token; the value is masked"
  session my-project-read 1h keyholder_run \
    '{"command":"curl","args":["-sS","-H","Authorization: Bearer {{vault://my-project/dev/API_TOKEN}}","{{vault://my-project/dev/API_URL}}"]}'
  note "Did the API receive the stored token? $(server_saw 'Bearer tok-fake-not-a-real-secret')"
}

main "$@"
