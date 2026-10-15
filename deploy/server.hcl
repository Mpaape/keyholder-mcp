# OpenBao server for Keyholder: one node, data on disk, TLS on localhost.
# The persistent vault guide (docs/server.md) explains each line; the server
# tutorial (demo/tutorials/server) runs this same file with the paths moved
# to a temporary directory.
#
#   bao server -config=deploy/server.hcl

# Integrated storage (Raft): the data survives restarts and can be backed up
# with `bao operator raft snapshot save`.
storage "raft" {
  path    = "/var/lib/openbao/data"
  node_id = "keyholder-1"
}

# Only this machine can reach the vault. The certificate must name
# 127.0.0.1; clients trust it through BAO_CACERT.
listener "tcp" {
  address       = "127.0.0.1:8200"
  tls_cert_file = "/etc/openbao/tls/tls.pem"
  tls_key_file  = "/etc/openbao/tls/tls.key"
}

api_addr     = "https://127.0.0.1:8200"
cluster_addr = "https://127.0.0.1:8201"

# Recommended with integrated storage, which keeps its data in memory-mapped
# files; also lets the server run in containers without IPC_LOCK.
disable_mlock = true
ui            = false
