# Cofre persistente

O servidor dev de [Começando](getting-started.md) guarda tudo em memória e não tem TLS. Para segredos de verdade, rode o OpenBao como servidor persistente. Esta página monta um nó na sua máquina, o cenário para o qual o Keyholder foi feito: uma pessoa, um cofre, o agente na mesma máquina.

Cada comando daqui também roda no [tutorial do cofre persistente](tutorials/server.md), no CI, com o mesmo [`deploy/server.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/server.hcl). As saídas estão lá.

## O que este cenário decide

| Questão | Escolha | Por quê |
| --- | --- | --- |
| Armazenamento | Armazenamento integrado (Raft), um nó | Dados em disco, sobrevivem a reinícios, backup com um comando; nenhum outro serviço para manter |
| Rede | TLS só em `127.0.0.1:8200` | Só esta máquina alcança o cofre; os comandos do agente rodam aqui também |
| Destrancamento (unseal) | Você, depois de cada reinício, com 2 de 3 chaves (Shamir) | Sem conta de nuvem e sem chave no mesmo disco dos dados; o cofre está trancado sempre que a máquina esteve desligada |
| Token root | Usado uma vez e revogado | Um login de administrador com política restrita faz o trabalho do dia a dia |

O **destrancamento automático** (um KMS de nuvem, um HSM ou o motor transit de outro OpenBao) é suportado pelo OpenBao e muda só um bloco `seal` na configuração: veja [seal concepts](https://openbao.org/docs/concepts/seal/). O Keyholder não o testa; ele faz sentido num servidor compartilhado, que não é este cenário.

No Windows, rode o servidor dentro do WSL ou de um container: veja [Windows e WSL](windows-wsl.md).

## 1. Arquivos e certificado

```bash
sudo mkdir -p /etc/openbao/tls /var/lib/openbao/data
sudo cp deploy/server.hcl /etc/openbao/server.hcl
```

O certificado precisa nomear `127.0.0.1`. Com uma autoridade certificadora da empresa, peça um a ela. Caso contrário, crie uma autoridade local e depois tire a chave dela da máquina:

```bash
cd /etc/openbao/tls
sudo openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -days 3650 \
  -subj /CN=keyholder-local-ca -keyout ca.key -out ca.pem
sudo openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -subj /CN=127.0.0.1 \
  -keyout tls.key -out tls.csr
printf 'subjectAltName=IP:127.0.0.1,DNS:localhost\n' | sudo tee ext >/dev/null
sudo openssl x509 -req -in tls.csr -CA ca.pem -CAkey ca.key -CAcreateserial -days 825 \
  -extfile ext -out tls.pem
```

Todo cliente do cofre, tanto o CLI `bao` quanto o Keyholder, confia nele por duas variáveis:

```bash
export BAO_ADDR=https://127.0.0.1:8200 BAO_CACERT=/etc/openbao/tls/ca.pem
```

## 2. Subir o servidor

```bash
bao server -config=/etc/openbao/server.hcl
```

Para subir junto com a máquina, basta uma unit do systemd. Esta é um exemplo; o tutorial sobe o servidor direto:

```ini
# /etc/systemd/system/openbao.service
[Unit]
Description=OpenBao for Keyholder
After=network.target

[Service]
ExecStart=/usr/local/bin/bao server -config=/etc/openbao/server.hcl
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

O servidor sempre sobe **trancado** (selado, *sealed*). Até ser destrancado, não tem chave nenhuma em memória e só responde o próprio status.

## 3. Inicializar uma vez

```bash
bao operator init -key-shares=3 -key-threshold=2
```

Isso imprime 3 chaves de destrancamento e o token root, uma única vez. Quaisquer 2 chaves destrancam o cofre; 1 sozinha não. Guarde as chaves em lugares diferentes: um gerenciador de senhas, uma cópia impressa numa gaveta, uma segunda pessoa. Se as 3 se perderem, os dados não podem ser recuperados.

Depois destranque, informando duas chaves diferentes:

```bash
bao operator unseal     # pede uma chave
bao operator unseal     # outra chave
```

## 4. Configurar e revogar o token root

Com o token root, uma vez:

```bash
export BAO_TOKEN=<token root>
bao secrets enable -path=secret kv-v2
bao policy write my-project-read deploy/policy-project-read.hcl
bao policy write keyholder-admin deploy/policy-admin.hcl
bao auth enable userpass
bao write auth/userpass/users/admin password=<senha> token_policies=keyholder-admin token_ttl=8h token_max_ttl=8h
bao token revoke -self
unset BAO_TOKEN
```

A [`policy-admin.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-admin.hcl) deixa o administrador guardar segredos, escrever políticas, emitir tokens e fazer backup, e nada mais. Se um dia precisar do token root de novo, gere um novo com as chaves de destrancamento (`bao operator generate-root`).

## 5. Todo dia

Entre, guarde o que o agente vai usar e emita o token embrulhado dele:

```bash
bao login -method=userpass username=admin
bao kv put -mount=secret my-project/dev API_TOKEN=...
export BAO_WRAPPING_TOKEN="$(bao token create -policy=my-project-read -ttl=8h -explicit-max-ttl=8h -wrap-ttl=5m -field=wrapping_token)"
```

Depois inicie seu cliente MCP com `BAO_ADDR`, `BAO_CACERT`, `KEYHOLDER_SCOPE` e `KEYHOLDER_ALLOW` definidos, como em [Clientes MCP](clients.md).

## 6. Depois de um reinício

O cofre volta trancado. O Keyholder avisa no `keyholder_status`, com `"sealed": true` e um erro de sessão que pede o destrancamento, e não consome o token embrulhado. Destranque com 2 chaves e reinicie o servidor MCP (ou reconecte no cliente) para ele abrir a sessão.

## 7. Backups

```bash
bao operator raft snapshot save keyholder-$(date +%F).snap
```

O snapshot é cifrado com as chaves do cofre. Restaure com `bao operator raft snapshot restore` e destranque com as mesmas chaves. Guarde snapshots e chaves de destrancamento separados.

## O que este cenário não cobre

- **Segundo fator** no login do administrador. O OpenBao suporta TOTP em logins; o Keyholder ainda não escolheu como usá-lo ([decisão 2](spec.md#9-decisoes-em-aberto)).
- **Cofre compartilhado** para um time, com alta disponibilidade e destrancamento automático. Veja a [documentação do OpenBao](https://openbao.org/docs/).
