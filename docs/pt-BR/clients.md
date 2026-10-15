# Clientes MCP

O `keyholder-mcp` fala MCP por **stdio**, então qualquer cliente MCP consegue iniciá-lo. Esta página mostra Claude Code, Cursor e Codex.

## Deixe o token fora dos arquivos de configuração

Todos os clientes abaixo conseguem ler o token embrulhado do ambiente do seu shell, em vez de um arquivo. Exporte-o no terminal que inicia o cliente:

```bash
export BAO_WRAPPING_TOKEN="$(bao token create -policy=my-project-read -ttl=8h -explicit-max-ttl=8h -wrap-ttl=5m -field=wrapping_token)"
```

Mesmo que acabe num arquivo, o token embrulhado é de uso único: depois que o `keyholder-mcp` o desembrulha, o valor não serve para nada. Cada partida do servidor precisa de um novo, e é assim que o acesso é renovado fora do chat.

## Claude Code

Pela CLI ([documentação oficial](https://code.claude.com/docs/en/mcp)):

```bash
claude mcp add \
  --env BAO_ADDR=http://127.0.0.1:8200 \
  --env KEYHOLDER_SCOPE=my-project \
  --env KEYHOLDER_ALLOW=curl,aws,gcloud \
  --transport stdio --scope project \
  keyholder -- keyholder-mcp
```

O nome do servidor não pode vir logo depois de um `--env`, senão a CLI o lê como outro `CHAVE=valor`; mantenha `--transport` ou `--scope` no meio. Com `--scope project` a entrada é gravada no `.mcp.json`; acrescente a linha `BAO_WRAPPING_TOKEN` mostrada abaixo para o token vir do seu ambiente.

No `.mcp.json` na raiz do projeto, o Claude Code expande `${VAR}` a partir do seu ambiente:

```json
{
  "mcpServers": {
    "keyholder": {
      "type": "stdio",
      "command": "keyholder-mcp",
      "env": {
        "BAO_ADDR": "http://127.0.0.1:8200",
        "BAO_WRAPPING_TOKEN": "${BAO_WRAPPING_TOKEN}",
        "KEYHOLDER_SCOPE": "my-project",
        "KEYHOLDER_ALLOW": "curl,aws,gcloud"
      }
    }
  }
}
```

Confira com `claude mcp get keyholder` ou `/mcp` dentro do Claude Code.

## Cursor

Em `.cursor/mcp.json` (projeto) ou `~/.cursor/mcp.json` (global) ([documentação oficial](https://cursor.com/docs/mcp)); o Cursor expande `${env:NOME}`:

```json
{
  "mcpServers": {
    "keyholder": {
      "type": "stdio",
      "command": "keyholder-mcp",
      "env": {
        "BAO_ADDR": "http://127.0.0.1:8200",
        "BAO_WRAPPING_TOKEN": "${env:BAO_WRAPPING_TOKEN}",
        "KEYHOLDER_SCOPE": "my-project",
        "KEYHOLDER_ALLOW": "curl,aws,gcloud"
      }
    }
  }
}
```

## Codex

Em `~/.codex/config.toml` ([documentação oficial](https://developers.openai.com/codex/mcp)); `env_vars` encaminha variáveis do seu ambiente:

```toml
[mcp_servers.keyholder]
command = "keyholder-mcp"
env_vars = ["BAO_WRAPPING_TOKEN"]

[mcp_servers.keyholder.env]
BAO_ADDR = "http://127.0.0.1:8200"
KEYHOLDER_SCOPE = "my-project"
KEYHOLDER_ALLOW = "curl,aws,gcloud"
```

Ou pela CLI:

```bash
codex mcp add keyholder --env BAO_ADDR=http://127.0.0.1:8200 --env KEYHOLDER_SCOPE=my-project \
  --env KEYHOLDER_ALLOW=curl,aws,gcloud -- keyholder-mcp
```

O Codex pergunta antes de cada chamada de tool MCP. O `codex exec` não tem como perguntar, então toda chamada falha com `MCP tool call requires approval, but approval policy is never`, a menos que as tools do servidor estejam aprovadas na configuração:

```toml
[mcp_servers.keyholder]
default_tools_approval_mode = "approve"
```

As proteções continuam valendo: quem decide o que roda são `KEYHOLDER_ALLOW` e `KEYHOLDER_SCOPE`, não a aprovação.

## Outros clientes

Qualquer cliente que inicie um servidor stdio com um comando e variáveis de ambiente funciona: o comando é `keyholder-mcp`, sem argumentos, e as variáveis estão em [Configuração](configuration.md#variaveis-de-ambiente).

## O que o agente vê

| Tool | Devolve |
| --- | --- |
| `keyholder_status` | Estado do cofre e da sessão, TTL, políticas; nunca o token |
| `keyholder_list` | Nomes de referências `vault://` no escopo |
| `keyholder_run` | Código de saída, stdout e stderr mascarados, referências usadas |

Entradas e saídas campo a campo: [Configuração → Tools](configuration.md#tools).

!!! note "Verificado"
    Os comandos e formatos de arquivo desta página vêm da documentação oficial de cada cliente, nos links acima. Os tutoriais exercitam o servidor com o `khcall`; os clientes em si não rodam no CI.
