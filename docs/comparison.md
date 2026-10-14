# How it compares

Keyholder is not the only MCP server in front of OpenBao or Vault. It is the only one we found built so that the model **never receives a secret value**. The others give the agent tools to read and manage the vault; Keyholder only lets the agent use what is in it.

Checked on 2026-10-14 against each project's README. "Not documented" means the README does not say, not that the feature is missing; open an issue if something here is out of date.

<div class="kh-compare" markdown>

| | Keyholder MCP | mcp-vault | openbao-mcp | vault-mcp-server |
| --- | --- | --- | --- | --- |
| Maintainer | [Mpaape](https://github.com/Mpaape/keyholder-mcp) | [Cloud Temple](https://github.com/Cloud-Temple/mcp-vault) | [Knuckles Team](https://github.com/Knuckles-Team/openbao-mcp) | [HashiCorp](https://github.com/hashicorp/vault-mcp-server) |
| Backend | OpenBao | OpenBao, embedded | OpenBao | HashiCorp Vault |
| License | MIT | Apache-2.0 | MIT | MPL-2.0 |
| Language | Go | Python | Python | Go |
| Model receives secret values | <span class="kh-yes">Never: no tool reads a value</span> | <span class="kh-no">Yes</span> (1) | <span class="kh-no">Has `openbao_read_secret`</span> (2) | <span class="kh-no">Yes</span> (3) |
| Value injected when a command runs | <span class="kh-yes">Yes</span> | <span class="kh-unk">Not documented</span> | <span class="kh-unk">Not documented</span> | <span class="kh-unk">Not documented</span> |
| Output masked | <span class="kh-yes">Yes, every resolved value</span> | <span class="kh-unk">Not documented</span> | <span class="kh-unk">Not documented</span> | <span class="kh-unk">Not documented</span> |
| Temporary access | <span class="kh-yes">Wrapped single-use token, TTL</span> | <span class="kh-yes">Single-use wrapping, SSH CA, PKI</span> | <span class="kh-unk">Not documented</span> | <span class="kh-unk">Not documented</span> |
| Scope of the agent | Run named commands with references from one project | 39 tools: secrets, SSH, PKI, policies, audit | OpenBao operations | Mounts, KV read and write, PKI |

</div>

1. The mcp-vault README states that an authorized secret necessarily exists in clear in the response handed to the authorized client.
2. The README describes the tool as "Read a secret key-value path" and does not say whether the value is returned.
3. The README warns that the server may expose Vault data, including secrets, to the MCP client and the LLM. It is meant for local use only.

## When to pick which

- **Pick Keyholder** when the agent should *use* credentials (call an API, run a cloud CLI) and the value must not reach the model, its transcript or its logs.
- **Pick one of the others** when the agent should *administer* the vault: create mounts, issue certificates, write policies. That work needs the agent to act on the vault directly, which Keyholder refuses by design.

The two models combine: an administration server for a trusted operator session, and Keyholder for everyday agent work.

## What "never sees it" means here

The guarantee is about the conversation with the model, and it has edges:

- The command Keyholder runs receives the value, as any real use of a credential does.
- Masking matches the exact value. A command that encodes it (base64, for example) escapes the mask; the [masking tutorial](tutorials/masking.md) shows it, and the allow list is how you prevent it.
- A hostile agent running as your own OS user can go around any MCP server.

The [security model](security.md) lists every limit.
