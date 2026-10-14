# Minimal policy for an agent to use one project's secrets in KV v2.
# Replace "my-project" with the value of KEYHOLDER_SCOPE and "secret" with the mount.
#
#   bao policy write my-project-read deploy/policy-project-read.hcl

# Read the values (used only by the runner, at execution time).
path "secret/data/my-project/*" {
  capabilities = ["read"]
}

# List the names (keyholder_list).
path "secret/metadata/my-project" {
  capabilities = ["list"]
}
path "secret/metadata/my-project/*" {
  capabilities = ["list", "read"]
}

# keyholder_status looks up its own token. OpenBao's "default" policy already
# grants this; it is only needed here when the token is created with
# -no-default-policy.
path "auth/token/lookup-self" {
  capabilities = ["read"]
}
