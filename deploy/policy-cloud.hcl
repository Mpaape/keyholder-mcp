# Example policy for an agent that also uses dynamic cloud credentials.
# Add it next to the project policy (policy-project-read.hcl) on the session
# token, and list the same paths in KEYHOLDER_DYNAMIC.
#
#   bao policy write my-project-cloud deploy/policy-cloud.hcl

# AWS secrets engine (plugin openbao-plugin-secrets-aws), role "deploy".
path "aws/creds/deploy" {
  capabilities = ["read"]
}

# Google Cloud secrets engine (plugin openbao-plugin-secrets-gcp), roleset "viewer".
path "gcp/roleset/viewer/token" {
  capabilities = ["read"]
}

# keyholder-mcp revokes each dynamic credential when the command ends.
path "sys/leases/revoke" {
  capabilities = ["update"]
}
