# Política mínima para um agente usar os segredos de um projeto no KV v2.
# Troque "meu-projeto" pelo valor de KEYHOLDER_SCOPE e "secret" pelo mount.
#
#   bao policy write meu-projeto-leitura deploy/policy-projeto-leitura.hcl

# Ler os valores (usado só pelo runner, na execução).
path "secret/data/meu-projeto/*" {
  capabilities = ["read"]
}

# Listar os nomes (keyholder_listar).
path "secret/metadata/meu-projeto" {
  capabilities = ["list"]
}
path "secret/metadata/meu-projeto/*" {
  capabilities = ["list", "read"]
}

# keyholder_status consulta o próprio token. A política "default" do OpenBao
# já concede isso; só é preciso aqui se o token for criado com -no-default-policy.
path "auth/token/lookup-self" {
  capabilities = ["read"]
}
