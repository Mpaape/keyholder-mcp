# Tutorial: credenciais de nuvem

## Objetivo

Ao final você verá o agente rodar a AWS CLI com uma credencial que o OpenBao cria para aquela execução e apaga quando o comando termina, e rodar o `gcloud` com um token de acesso que ele nunca vê.

Cada comando e saída abaixo vem de uma execução real, gravada em `demo/tutorials/cloud/expected/` e conferida pelo `run.sh --check`.

**O que é real e o que é emulado.** O OpenBao e o [plugin de segredos da AWS](https://github.com/openbao/openbao-plugins/tree/main/secrets/aws) são os de verdade. A AWS em si é o [moto](https://github.com/getmoto/moto), um emulador local das APIs de IAM e STS, então a execução não precisa de conta AWS. O plugin do Google Cloud precisa de um projeto Google real, então o caso do gcloud guarda um token falso no KV e mostra o mecanismo de arquivo que as credenciais `gcp:` usam.

## Rodar

```bash
pip install "moto[server]"     # mais as CLIs aws e gcloud
export KH_AWS_PLUGIN=/caminho/para/openbao-plugin-secrets-aws   # da release do plugin no GitHub
bash demo/tutorials/cloud/run.sh all
bash demo/tutorials/cloud/run.sh --check
```

Script: [`demo/tutorials/cloud/run.sh`](https://github.com/Mpaape/keyholder-mcp/blob/main/demo/tutorials/cloud/run.sh). A sessão usa duas políticas: [`policy-project-read.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-project-read.hcl) e [`policy-cloud.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-cloud.hcl). `KEYHOLDER_DYNAMIC=aws/creds/deploy`.

## 1. O humano habilita o engine da AWS

Fora do chat. O role cria um usuário IAM por credencial, que só pode chamar `sts:GetCallerIdentity`.

```text
--8<-- "demo/tutorials/cloud/expected/01-engine.txt"
```

## 2. O agente vê o caminho, não uma credencial

O `keyholder_list` mostra o caminho dinâmico permitido e como usá-lo. A listagem nunca o lê: uma leitura criaria uma credencial.

```text
--8<-- "demo/tutorials/cloud/expected/02-list.txt"
```

## 3. Uma credencial por execução

O agente pede `credentials: ["aws:aws/creds/deploy"]`. O Keyholder lê o caminho uma vez, define `AWS_ACCESS_KEY_ID` e `AWS_SECRET_ACCESS_KEY` no filho, roda o `aws` e revoga o lease quando ele termina. O ARN mostra o usuário que o OpenBao criou para essa execução; depois dela não sobra nenhum.

```text
--8<-- "demo/tutorials/cloud/expected/03-aws.txt"
```

## 4. Só caminhos listados

Um caminho fora de `KEYHOLDER_DYNAMIC` é recusado antes de qualquer leitura, mesmo que a política permitisse.

```text
--8<-- "demo/tutorials/cloud/expected/04-not-allowed.txt"
```

## 5. gcloud com arquivo de token

O `files` entrega ao `gcloud` o caminho de um arquivo temporário com o token (`CLOUDSDK_AUTH_ACCESS_TOKEN_FILE`), apagado depois da execução. O gcloud imprime o token que recebeu; o agente o lê mascarado. Com o plugin do Google Cloud, `credentials: ["gcp:gcp/roleset/<nome>/token"]` faz o mesmo com um token criado pelo plugin.

```text
--8<-- "demo/tutorials/cloud/expected/05-gcloud.txt"
```

## O que isso mostra

- Uma credencial dinâmica existe só durante o comando: lida na execução, revogada depois dela.
- O agente vê o caminho e o resultado, nunca as chaves nem o token.
- A revogação depende do tipo de credencial: usuários IAM são apagados; credenciais STS e tokens de acesso do GCP só expiram. Veja [Configuração → Credenciais dinâmicas](../configuration.md#credenciais-dinamicas).
