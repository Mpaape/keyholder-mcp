# Day-to-day administrator of a Keyholder vault: stores secrets, writes
# policies and issues the agents' wrapped tokens. Use it instead of the root
# token, which is revoked after setup (docs/server.md).
#
#   bao policy write keyholder-admin deploy/policy-admin.hcl

# Store, read and remove secrets in the KV v2 mount "secret".
path "secret/data/*" {
  capabilities = ["create", "update", "read", "delete"]
}
path "secret/metadata/*" {
  capabilities = ["list", "read", "delete"]
}

# Write the policies that tokens are issued with.
path "sys/policies/acl/*" {
  capabilities = ["create", "update", "read", "list"]
}

# Issue tokens with policies the administrator does not hold itself.
path "auth/token/create" {
  capabilities = ["create", "update", "sudo"]
}

# Back up the vault.
path "sys/storage/raft/snapshot" {
  capabilities = ["read"]
}
