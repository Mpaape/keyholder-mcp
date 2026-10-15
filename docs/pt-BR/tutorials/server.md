# Tutorial: cofre persistente

## Objetivo

Ao final você vai ver um servidor OpenBao que guarda os dados em disco e fala TLS. Ele é inicializado uma vez, destrancado com 2 de 3 chaves e configurado sem guardar o token root. O Keyholder o usa por TLS, um reinício o tranca de novo, ele é destrancado e por fim ganha um backup.

Cada comando e cada saída abaixo vêm de uma execução real, gravada em `demo/tutorials/server/expected/` e conferida por `run.sh --check`. As saídas são do programa, em inglês. Os passos, e o motivo de cada um, estão no [guia do cofre persistente](../server.md).

**O que muda em relação a uma instalação real.** O servidor roda o próprio [`deploy/server.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/server.hcl), com os caminhos levados para um diretório temporário e a porta trocada para 18200. A autoridade certificadora é criada para a execução. As chaves de destrancamento, o token root e a senha do administrador ficam nesse diretório temporário e são apagados com ele.

## Como rodar

```bash
bash demo/tutorials/server/run.sh all       # precisa de bao, openssl, python3, curl e go
bash demo/tutorials/server/run.sh --check
```

Script: [`demo/tutorials/server/run.sh`](https://github.com/Mpaape/keyholder-mcp/blob/main/demo/tutorials/server/run.sh). Políticas: [`policy-project-read.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-project-read.hcl) e [`policy-admin.hcl`](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-admin.hcl).

## 1. Inicializar uma vez, destrancar com 2 de 3 chaves

Um servidor novo vem trancado e sem inicializar. O `bao operator init` roda uma única vez na vida do cofre e imprime as chaves de destrancamento e o token root; aqui elas vão para um arquivo que nunca entra no repositório. Cada chave avança o progresso em um; a segunda abre o cofre.

```text
--8<-- "demo/tutorials/server/expected/01-init.txt"
```

## 2. Configurar e revogar o token root

O token root cria o mount KV, as políticas e um login de administrador, e depois é revogado. A partir daí o administrador entra com senha e recebe um token de 8 horas, que guarda segredos e emite os tokens embrulhados dos agentes, mas não muda a configuração do cofre.

```text
--8<-- "demo/tutorials/server/expected/02-setup.txt"
```

## 3. Keyholder por TLS

Com `BAO_CACERT` apontando para a autoridade certificadora, a sessão abre e a API recebe o token. Sem ele, o Keyholder recusa a conexão e diz qual variável definir.

```text
--8<-- "demo/tutorials/server/expected/03-tls.txt"
```

## 4. Um reinício tranca o cofre

Um token embrulhado é emitido e o servidor reinicia. Ele volta trancado: o Keyholder avisa e não consome o token. Depois de duas chaves, um novo início do Keyholder abre a sessão com esse mesmo token, e o segredo guardado antes do reinício continua lá.

```text
--8<-- "demo/tutorials/server/expected/04-restart.txt"
```

## 5. Backup

O administrador salva um snapshot do cofre inteiro. Ele é cifrado com as chaves do cofre, então é restaurado e destrancado com as mesmas chaves.

```text
--8<-- "demo/tutorials/server/expected/05-backup.txt"
```
