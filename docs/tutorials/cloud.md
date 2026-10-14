# Tutorial: cloud credentials

## Goal

At the end you will see the agent run the AWS CLI with a credential that OpenBao creates for that one run and deletes when the command ends, and run `gcloud` with an access token it never sees.

Every command and output below comes from a real run, recorded in `demo/tutorials/cloud/expected/` and checked by `run.sh --check`.

**What is real and what is emulated.** OpenBao and its [AWS secrets plugin](https://github.com/openbao/openbao-plugins/tree/main/secrets/aws) are the real ones. AWS itself is [moto](https://github.com/getmoto/moto), a local emulator of the IAM and STS APIs, so the run needs no AWS account. The Google Cloud plugin needs a real Google project, so the gcloud case keeps a fake token in the KV and shows the file mechanism that `gcp:` credentials use.

## Run it

```bash
pip install "moto[server]"     # plus the aws and gcloud CLIs
export KH_AWS_PLUGIN=/path/to/openbao-plugin-secrets-aws   # from the plugin's GitHub release
bash demo/tutorials/cloud/run.sh all
bash demo/tutorials/cloud/run.sh --check
```

Script: [`demo/tutorials/cloud/run.sh`](https://github.com/Mpaape/keyholder-mcp/blob/main/demo/tutorials/cloud/run.sh). The session uses two policies: [`policy-project-read.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-project-read.hcl) and [`policy-cloud.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-cloud.hcl). `KEYHOLDER_DYNAMIC=aws/creds/deploy`.

## 1. The human enables the AWS engine

Outside the chat. The role creates an IAM user per credential, allowed only to call `sts:GetCallerIdentity`.

```text
--8<-- "demo/tutorials/cloud/expected/01-engine.txt"
```

## 2. The agent sees the path, not a credential

`keyholder_list` shows the allowed dynamic path and how to use it. Listing never reads it: a read would create a credential.

```text
--8<-- "demo/tutorials/cloud/expected/02-list.txt"
```

## 3. A credential for one run

The agent asks for `credentials: ["aws:aws/creds/deploy"]`. Keyholder reads the path once, sets `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` for the child, runs `aws`, and revokes the lease when it exits. The ARN shows the user OpenBao created for this run; after the run no user is left.

```text
--8<-- "demo/tutorials/cloud/expected/03-aws.txt"
```

## 4. Only listed paths

A path outside `KEYHOLDER_DYNAMIC` is refused before anything is read, even when the policy would allow it.

```text
--8<-- "demo/tutorials/cloud/expected/04-not-allowed.txt"
```

## 5. gcloud with an access token file

`files` gives `gcloud` the path of a temporary file holding the token (`CLOUDSDK_AUTH_ACCESS_TOKEN_FILE`), deleted after the run. gcloud prints the token it was given; the agent reads it masked. With the Google Cloud plugin, `credentials: ["gcp:gcp/roleset/<name>/token"]` does the same with a token the plugin creates.

```text
--8<-- "demo/tutorials/cloud/expected/05-gcloud.txt"
```

## What this shows

- A dynamic credential exists only for the command: read at execution, revoked after it.
- The agent sees the path and the result, never the keys or the token.
- Revocation depends on the credential type: IAM users are deleted; STS credentials and GCP access tokens can only expire. See [Configuration → Dynamic credentials](../configuration.md#dynamic-credentials).
