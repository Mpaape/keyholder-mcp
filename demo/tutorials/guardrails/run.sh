#!/usr/bin/env bash
# Tutorial: guardrails. The project scope, the command allow list and the
# environment filter, each refusing what it should.
#   bash demo/tutorials/guardrails/run.sh all | --check
source "$(dirname "$0")/../lib.sh"

# printenv is allowed here ONLY to prove that the vault token does not reach
# the child process. Never allow printenv, env, sh, bash or python in real use.
export KEYHOLDER_SCOPE=my-project KEYHOLDER_ALLOW=curl,printenv
CASES=(01-scope 02-allow-list 03-reserved-env 04-child-env)

setup() {
  admin kv put -mount=secret my-project/dev API_TOKEN=tok-fake-0123456789 >/dev/null
  admin kv put -mount=secret other-project/prod DB_PASSWORD=pw-fake-other >/dev/null
  # A broad policy on purpose: the scope check must hold even when the
  # vault would allow the read.
  printf 'path "secret/*" { capabilities = ["read", "list"] }\n' | admin policy write broad-read - >/dev/null
}

case_01-scope() {
  step 01-scope "Only the project scope is listed and usable, even with a broad policy"
  session broad-read 1h keyholder_list \
    keyholder_run '{"command":"curl","args":["-sS","-H","X-Pass: {{vault://other-project/prod/DB_PASSWORD}}","http://127.0.0.1:9/"]}'
}

case_02-allow-list() {
  step 02-allow-list "Commands outside the list, with a path or with a placeholder are refused"
  session broad-read 1h \
    keyholder_run '{"command":"sh","args":["-c","echo hi"]}' \
    keyholder_run '{"command":"/usr/bin/curl","args":["--version"]}' \
    keyholder_run '{"command":"{{vault://my-project/dev/API_TOKEN}}"}'
}

case_03-reserved-env() {
  step 03-reserved-env "The agent cannot set BAO_*, VAULT_* or KEYHOLDER_* variables"
  session broad-read 1h keyholder_run '{"command":"curl","args":["--version"],"env":{"BAO_ADDR":"http://attacker.example.com"}}'
}

case_04-child-env() {
  step 04-child-env "The vault token never reaches the child process"
  note "The server itself was started with BAO_TOKEN set below; the child must not see it."
  local w; w="$(wrapped_token broad-read 1h)"
  show_as "BAO_TOKEN=<root token> BAO_WRAPPING_TOKEN=<wrapped token> khcall keyholder_run '{\"command\":\"printenv\",\"args\":[\"BAO_TOKEN\"]}'" \
    env BAO_TOKEN="$ROOT_TOKEN" BAO_WRAPPING_TOKEN="$w" khcall keyholder_run '{"command":"printenv","args":["BAO_TOKEN"]}'
  note "printenv exits 1 and prints nothing: BAO_TOKEN is not set in the child process."
}

main "$@"
