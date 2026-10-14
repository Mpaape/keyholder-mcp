# Keyholder MCP

Cofre local de credenciais para agentes de IA. O agente descobre **nomes** de credenciais e as usa na execução, mas o **valor nunca entra no chat**. O acesso é temporário: expira sozinho.

> Status: prova de conceito. Servidor MCP em Go sobre o [OpenBao](https://openbao.org), testado de ponta a ponta contra `bao server -dev` 2.7.1. Spec completo em [`docs/SPEC.md`](docs/SPEC.md).

## Como funciona

1. Os segredos ficam no KV v2 do OpenBao, um caminho por projeto (`secret/meu-projeto/dev`).
2. Você emite fora do chat um token **embrulhado** (uso único) com prazo e política mínima.
3. O `keyholder-mcp` desembrulha esse token ao iniciar e o guarda só em memória.
4. O agente lista as referências (`vault://meu-projeto/dev/API_TOKEN`) e pede execuções com placeholders:
   `curl -H "Authorization: Bearer {{vault://meu-projeto/dev/API_TOKEN}}" https://api.exemplo.com/health`
5. O runner troca o placeholder pelo valor só na hora de executar, sem shell, e devolve a saída com o valor mascarado.
6. Quando o token expira, as tools falham com uma mensagem clara e você renova fora do chat.

## Tools MCP

| Tool | O que faz | Devolve |
| --- | --- | --- |
| `keyholder_status` | Estado do cofre e da sessão | Selado ou não, TTL restante, políticas; nunca o token |
| `keyholder_listar` | Referências do escopo do projeto | Só nomes `vault://...` |
| `keyholder_executar` | Roda um comando permitido com `{{vault://...}}` em `args` ou `env` | Código de saída, stdout e stderr mascarados, referências usadas |

## Compilar

Requer Go 1.25 ou superior.

```sh
go mod tidy        # gera o go.sum, verificado pelo checksum database do Go
go test ./...
go build -o keyholder-mcp ./cmd/keyholder-mcp     # no Windows: -o keyholder-mcp.exe
```

## Prova de conceito (OpenBao dev, dados em memória)

Num terminal (WSL ou Linux):

```sh
bao server -dev -dev-root-token-id=root-dev        # NUNCA use o modo dev com segredos reais
```

Em outro:

```sh
export BAO_ADDR=http://127.0.0.1:8200 BAO_TOKEN=root-dev
bao kv put secret/meu-projeto/dev API_TOKEN=valor-de-teste
bao policy write meu-projeto-leitura deploy/policy-projeto-leitura.hcl

# Token de 1 hora, embrulhado: o valor impresso só funciona uma vez, por 5 minutos.
bao token create -policy=meu-projeto-leitura -ttl=1h -explicit-max-ttl=1h -wrap-ttl=5m -field=wrapping_token
```

## Configuração do cliente MCP

O formato abaixo é o `mcpServers` usado por vários clientes; confira a documentação do seu.

```json
{
  "mcpServers": {
    "keyholder": {
      "command": "/caminho/para/keyholder-mcp",
      "env": {
        "BAO_ADDR": "http://127.0.0.1:8200",
        "BAO_WRAPPING_TOKEN": "<valor impresso pelo comando acima>",
        "KEYHOLDER_SCOPE": "meu-projeto",
        "KEYHOLDER_ALLOW": "curl,aws,gcloud"
      }
    }
  }
}
```

O token embrulhado fica no arquivo de configuração, mas deixa de valer assim que o servidor o desembrulha. Cada reinício do servidor precisa de um token embrulhado novo, que é a renovação fora do chat.

| Variável | Padrão | Uso |
| --- | --- | --- |
| `BAO_ADDR` | do cliente OpenBao | Endereço do cofre |
| `BAO_WRAPPING_TOKEN` | | Token embrulhado, de uso único (preferido) |
| `BAO_TOKEN` | | Token comum; aceito só para testes |
| `KEYHOLDER_KV_MOUNT` | `secret` | Mount do KV v2 |
| `KEYHOLDER_SCOPE` | vazio (tudo) | Prefixo do projeto; referências fora dele são recusadas |
| `KEYHOLDER_ALLOW` | vazio (nada) | Comandos permitidos, separados por vírgula |
| `KEYHOLDER_TIMEOUT` | `5m` | Tempo máximo por execução |

## Garantias e limites

- O valor não passa pelas mensagens MCP, mas o processo executado o recebe, como em qualquer uso real de credencial.
- O mascaramento casa só o valor exato. Um comando que codifique o segredo (base64, por exemplo) o expõe. Por isso a lista de comandos permitidos importa: não libere `sh`, `bash`, `python` ou `env`.
- `BAO_*`, `VAULT_*` e `KEYHOLDER_*` são removidas do ambiente do processo filho, então o token do cofre não chega ao comando.
- Não protege contra um agente hostil que roda com o seu usuário do sistema.

## Licença

Ainda não definida (decisão aberta no spec).
