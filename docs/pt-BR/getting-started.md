# Começando

Esta página leva você do zero a um agente chamando uma API com uma credencial que ele nunca vê. Cada passo aponta para a referência ou o tutorial que o explica a fundo.

## 1. Requisitos

| Ferramenta | Versão | Para quê |
| --- | --- | --- |
| [Go](https://go.dev/dl/) | 1.25 ou superior | Compilar o `keyholder-mcp` |
| [OpenBao](https://openbao.org/docs/install/) (`bao`) | 2.7.x testado | O cofre |
| Um cliente MCP | Claude Code, Cursor, Codex... | O agente |

## 2. Compilar

```bash
git clone https://github.com/Mpaape/keyholder-mcp.git
cd keyholder-mcp
go mod tidy                # gera o go.sum, conferido pelo checksum database do Go
go test ./...
go build -o keyholder-mcp ./cmd/keyholder-mcp     # Windows: -o keyholder-mcp.exe
go build -o khcall ./cmd/khcall                   # cliente opcional para depurar
./keyholder-mcp -version
```

Coloque os dois binários num diretório do seu `PATH`.

## 3. Subir um cofre

Para experimentar, o servidor **dev** do OpenBao basta. Ele guarda tudo em memória, escuta em `127.0.0.1:8200` sem TLS e perde tudo ao parar: nunca use com segredos reais.

```bash
bao server -dev -dev-root-token-id=root-dev
```

Para uso real, rode um OpenBao persistente com TLS e uma política de unseal; veja a [documentação do OpenBao](https://openbao.org/docs/) e [Windows e WSL](windows-wsl.md) para onde rodá-lo.

## 4. Guardar um segredo e uma política

Em outro terminal, como administrador do cofre:

```bash
export BAO_ADDR=http://127.0.0.1:8200 BAO_TOKEN=root-dev
bao kv put -mount=secret my-project/dev API_TOKEN=sua-chave-de-api
bao policy write my-project-read deploy/policy-project-read.hcl
```

A [política](https://github.com/Mpaape/keyholder-mcp/blob/main/deploy/policy-project-read.hcl) só permite ler e listar `secret/my-project/*`.

## 5. Emitir um token embrulhado

```bash
bao token create -policy=my-project-read -ttl=8h -explicit-max-ttl=8h -wrap-ttl=5m -field=wrapping_token
```

O valor impresso é um token **embrulhado**: pode ser desembrulhado uma vez, em até 5 minutos, e o token de dentro dura no máximo 8 horas. Entregue-o ao `keyholder-mcp` por `BAO_WRAPPING_TOKEN`. Por que isso é mais seguro: [Acesso temporário](tutorials/access.md).

## 6. Ligar o agente

Adicione o `keyholder-mcp` ao seu cliente MCP com estas variáveis. Passos exatos para Claude Code, Cursor e Codex: [Clientes MCP](clients.md).

| Variável | Exemplo |
| --- | --- |
| `BAO_ADDR` | `http://127.0.0.1:8200` |
| `BAO_WRAPPING_TOKEN` | o valor do passo 5 |
| `KEYHOLDER_SCOPE` | `my-project` |
| `KEYHOLDER_ALLOW` | `curl,aws,gcloud` |

## 7. Pedir ao agente

> Liste as credenciais que você tem e chame https://api.example.com/health com o token da API.

O agente chama `keyholder_list`, vê `vault://my-project/dev/API_TOKEN` e chama `keyholder_run` com `{{vault://my-project/dev/API_TOKEN}}` num cabeçalho. A resposta que ele lê vem com o valor mascarado.

## Conferir sem agente

O `khcall` é um cliente MCP mínimo para depuração. Ele sobe o servidor, chama tools numa sessão e imprime JSON:

```bash
export BAO_ADDR=http://127.0.0.1:8200 KEYHOLDER_SCOPE=my-project KEYHOLDER_ALLOW=curl
BAO_WRAPPING_TOKEN=<token embrulhado> khcall keyholder_status keyholder_list
```

O [tutorial de início rápido](tutorials/quickstart.md) mostra a saída real completa.

## Próximos passos

- [Tutoriais](tutorials/index.md): cada comportamento, com saídas reais.
- [Configuração](configuration.md): cada variável, tool e campo.
- [Modelo de segurança](security.md): o que é protegido e o que não é.
