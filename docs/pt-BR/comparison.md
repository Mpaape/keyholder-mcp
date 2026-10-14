# Comparação

O Keyholder não é o único servidor MCP na frente do OpenBao ou do Vault. É o único que encontramos feito para que o modelo **nunca receba o valor de um segredo**. Os outros dão ao agente tools para ler e administrar o cofre; o Keyholder só deixa o agente usar o que está nele.

Conferido em 2026-10-14 no README de cada projeto. "Não documentado" quer dizer que o README não diz, não que o recurso falte; abra uma issue se algo aqui estiver desatualizado.

<div class="kh-compare" markdown>

| | Keyholder MCP | mcp-vault | openbao-mcp | vault-mcp-server |
| --- | --- | --- | --- | --- |
| Mantido por | [Mpaape](https://github.com/Mpaape/keyholder-mcp) | [Cloud Temple](https://github.com/Cloud-Temple/mcp-vault) | [Knuckles Team](https://github.com/Knuckles-Team/openbao-mcp) | [HashiCorp](https://github.com/hashicorp/vault-mcp-server) |
| Cofre | OpenBao | OpenBao, embutido | OpenBao | HashiCorp Vault |
| Licença | MIT | Apache-2.0 | MIT | MPL-2.0 |
| Linguagem | Go | Python | Python | Go |
| O modelo recebe valores | <span class="kh-yes">Nunca: nenhuma tool lê um valor</span> | <span class="kh-no">Sim</span> (1) | <span class="kh-no">Tem `openbao_read_secret`</span> (2) | <span class="kh-no">Sim</span> (3) |
| Valor injetado quando o comando roda | <span class="kh-yes">Sim</span> | <span class="kh-unk">Não documentado</span> | <span class="kh-unk">Não documentado</span> | <span class="kh-unk">Não documentado</span> |
| Saída mascarada | <span class="kh-yes">Sim, todo valor resolvido</span> | <span class="kh-unk">Não documentado</span> | <span class="kh-unk">Não documentado</span> | <span class="kh-unk">Não documentado</span> |
| Credencial de nuvem por execução | <span class="kh-yes">AWS e GCP pelos plugins do OpenBao; usuários IAM da AWS revogados depois da execução</span> | <span class="kh-unk">Não documentado</span> | <span class="kh-unk">Não documentado</span> | <span class="kh-unk">Não documentado</span> |
| Acesso temporário | <span class="kh-yes">Token embrulhado de uso único, TTL</span> | <span class="kh-yes">Wrapping de uso único, CA de SSH, PKI</span> | <span class="kh-unk">Não documentado</span> | <span class="kh-unk">Não documentado</span> |
| Alcance do agente | Rodar comandos nomeados com referências de um projeto | 39 tools: segredos, SSH, PKI, políticas, auditoria | Operações do OpenBao | Mounts, leitura e escrita no KV, PKI |

</div>

1. O README do mcp-vault afirma que um segredo autorizado existe necessariamente em claro na resposta entregue ao cliente autorizado.
2. O README descreve a tool como "Read a secret key-value path" e não diz se o valor é devolvido.
3. O README avisa que o servidor pode expor dados do Vault, inclusive segredos, ao cliente MCP e ao LLM. É feito só para uso local.

## Quando escolher qual

- **Escolha o Keyholder** quando o agente deve *usar* credenciais (chamar uma API, rodar uma CLI de nuvem) e o valor não pode chegar ao modelo, à transcrição nem aos logs.
- **Escolha um dos outros** quando o agente deve *administrar* o cofre: criar mounts, emitir certificados, escrever políticas. Esse trabalho exige que o agente aja direto no cofre, o que o Keyholder recusa por desenho.

Os dois modelos se combinam: um servidor de administração para uma sessão de operador confiável, e o Keyholder para o trabalho diário do agente.

## O que "nunca vê" quer dizer aqui

A garantia é sobre a conversa com o modelo, e tem bordas:

- O comando que o Keyholder roda recebe o valor, como em qualquer uso real de credencial.
- O mascaramento casa o valor exato. Um comando que o codifique (base64, por exemplo) escapa; o [tutorial de mascaramento](tutorials/masking.md) mostra isso, e a lista de comandos permitidos é como você evita.
- Um agente hostil rodando com o seu usuário do sistema contorna qualquer servidor MCP.

O [modelo de segurança](security.md) lista todos os limites.
