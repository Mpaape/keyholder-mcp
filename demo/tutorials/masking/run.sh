#!/usr/bin/env bash
# Tutorial: masking and its limit. The value is hidden in stdout and stderr;
# an encoded value is not. This is why the allow list matters.
#   bash demo/tutorials/masking/run.sh all | --check
source "$(dirname "$0")/../lib.sh"

# python3 is allowed here ONLY to demonstrate the limit. Never allow an
# interpreter in real use.
export KEYHOLDER_SCOPE=my-project KEYHOLDER_ALLOW=curl,python3
CASES=(01-reflected 02-limit-encoded)

setup() {
  echo_server
  admin kv put -mount=secret my-project/dev API_TOKEN=tok-fake-not-a-real-secret >/dev/null
  admin policy write my-project-read deploy/policy-project-read.hcl >/dev/null
}

case_01-reflected() {
  step 01-reflected "The API reflects the header back: the agent sees the reference, not the value"
  session my-project-read 1h keyholder_run \
    '{"command":"curl","args":["-sS","-H","Authorization: Bearer {{vault://my-project/dev/API_TOKEN}}","http://127.0.0.1:18080/"]}'
  note "Did the API receive the stored token? $(server_saw 'Bearer tok-fake-not-a-real-secret')"
  note "stderr is masked the same way (covered by internal/runner tests)."
}

case_02-limit-encoded() {
  step 02-limit-encoded "LIMIT: an allowed interpreter can encode the value and escape the mask"
  session my-project-read 1h keyholder_run \
    '{"command":"python3","args":["-c","import base64,sys; print(base64.b64encode(sys.argv[1].encode()).decode())","{{vault://my-project/dev/API_TOKEN}}"]}'
  # Decode what the agent actually received, from the recorded stdout.
  note "Decoded: $(python3 -c 'import base64,json,sys; print(base64.b64decode(json.load(sys.stdin)["stdout"]).decode())' <"$WORK/stdout") (the fake token, unmasked)"
}

main "$@"
