#!/usr/bin/env bash
# Tutorial: cloud credentials. The agent runs the AWS CLI with a credential
# created for that one run and revoked after it, and runs gcloud with an
# access token it never sees.
#
# AWS is emulated locally with moto (pip install "moto[server]"); OpenBao
# uses its real AWS secrets plugin. Requirements on top of lib.sh: aws,
# gcloud, moto_server, and KH_AWS_PLUGIN pointing at the
# openbao-plugin-secrets-aws binary for this system.
#   bash demo/tutorials/cloud/run.sh all | --check
source "$(dirname "$0")/../lib.sh"

export KEYHOLDER_SCOPE=my-project KEYHOLDER_ALLOW=aws,gcloud
export KEYHOLDER_DYNAMIC=aws/creds/deploy
CASES=(01-engine 02-list 03-aws 04-not-allowed 05-gcloud)
MOTO=http://127.0.0.1:15000

# emulator <aws args...>: the human looks inside the emulator with its own
# (fake) root keys, never with the agent's.
emulator() {
  AWS_ACCESS_KEY_ID=moto-root AWS_SECRET_ACCESS_KEY=moto-root-secret AWS_DEFAULT_REGION=us-east-1 \
    aws --endpoint-url "$MOTO" "$@"
}

# The dev server registers the plugins found in this directory at start.
if [ "${1:-}" != --check ]; then
  [ -x "${KH_AWS_PLUGIN:-}" ] || { echo "set KH_AWS_PLUGIN to the openbao-plugin-secrets-aws binary"; exit 1; }
  mkdir -p "$WORK/plugins"
  cp "$KH_AWS_PLUGIN" "$WORK/plugins/openbao-plugin-secrets-aws"
fi
VAULT_ARGS=(-dev-plugin-dir="$WORK/plugins")

setup() {
  moto_server -p 15000 >"$WORK/moto.log" 2>&1 &
  PIDS+=($!)
  for _ in $(seq 1 50); do curl -s -o /dev/null "$MOTO/" && break; sleep 0.2; done
  admin kv put -mount=secret my-project/gcp ACCESS_TOKEN=tok-fake-gcp-access-token >/dev/null
  admin policy write my-project-read deploy/policy-project-read.hcl >/dev/null
  admin policy write my-project-cloud deploy/policy-cloud.hcl >/dev/null
}

# session with both policies: the project's KV and the cloud paths.
cloud_session() {
  local w
  w="$(admin token create -policy=my-project-read -policy=my-project-cloud -ttl=1h -explicit-max-ttl=1h -wrap-ttl=5m -field=wrapping_token)"
  show_as "BAO_WRAPPING_TOKEN=<wrapped token> khcall $(quote "$@")" env BAO_WRAPPING_TOKEN="$w" khcall "$@"
}

case_01-engine() {
  step 01-engine "The human enables the AWS secrets engine and a role (outside the chat)"
  show admin secrets enable -path=aws openbao-plugin-secrets-aws
  show admin write aws/config/root access_key=moto-root secret_key=moto-root-secret region=us-east-1 \
    iam_endpoint="$MOTO" sts_endpoint="$MOTO"
  show admin write aws/roles/deploy credential_type=iam_user \
    policy_document='{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"sts:GetCallerIdentity","Resource":"*"}]}'
  note "The emulator's root keys are fake. With real AWS they are an IAM user that may create users; the agent never gets them."
}

case_02-list() {
  step 02-list "The agent sees the dynamic path and how to use it, not a credential"
  cloud_session keyholder_list
}

case_03-aws() {
  step 03-aws "The agent runs the AWS CLI with a credential made for this run"
  cloud_session keyholder_run '{"command":"aws","args":["--endpoint-url","http://127.0.0.1:15000","sts","get-caller-identity","--query","Arn","--output","text"],"env":{"AWS_DEFAULT_REGION":"us-east-1"},"credentials":["aws:aws/creds/deploy"]}'
  note "IAM users left in AWS after the run (checked by the human): $(emulator iam list-users --query 'length(Users)' --output text)"
  note "Keyholder revoked the lease when aws exited, and OpenBao deleted the user it had created for the run."
}

case_04-not-allowed() {
  step 04-not-allowed "A dynamic path outside KEYHOLDER_DYNAMIC is refused before any read"
  cloud_session keyholder_run '{"command":"aws","args":["sts","get-caller-identity"],"credentials":["aws:aws/creds/admin"]}'
}

case_05-gcloud() {
  step 05-gcloud "gcloud reads an access token from a file it gets for this run only"
  # The GCP secrets plugin needs a real Google project, so this case keeps a
  # fake token in the KV. "credentials": ["gcp:gcp/roleset/<name>/token"]
  # does the same with a token the plugin creates.
  cloud_session keyholder_run '{"command":"gcloud","args":["auth","print-access-token"],"env":{"CLOUDSDK_CONFIG":"/tmp/keyholder-tutorial-gcloud","CLOUDSDK_CORE_DISABLE_PROMPTS":"1"},"files":{"CLOUDSDK_AUTH_ACCESS_TOKEN_FILE":"{{vault://my-project/gcp/ACCESS_TOKEN}}"}}'
}

main "$@"
