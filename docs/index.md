---
title: Keyholder MCP
hide:
  - navigation
  - toc
---

<div class="kh-home" markdown>

# Keyholder MCP

<div class="kh-hero" markdown>

<div markdown>

<p class="kh-hero__title">Your AI agent uses the credential. It never sees it.</p>

<p class="kh-hero__lede">Keyholder MCP is an <a href="https://modelcontextprotocol.io">MCP</a> server over <a href="https://openbao.org">OpenBao</a>. The agent writes a reference; Keyholder puts the real value into the command only when it runs, and hands the output back with the value masked. Access ends on its own.</p>

[Get started](getting-started.md){ .md-button .md-button--primary } [Download](https://github.com/Mpaape/keyholder-mcp/releases){ .md-button }

</div>

<div class="kh-demo" aria-label="One request seen by the API and by the agent">
  <div class="kh-demo__pane">
    <p class="kh-demo__who">The agent asks keyholder_run</p>
<pre class="kh-demo__code">curl -sS \
  -H "Authorization: Bearer <span class="kh-ref">{{vault://my-project/dev/API_TOKEN}}</span>" \
  <span class="kh-ref">{{vault://my-project/dev/API_URL}}</span></pre>
  </div>
  <div class="kh-demo__fork" aria-hidden="true"></div>
  <div class="kh-demo__split">
    <div class="kh-demo__pane kh-demo__pane--api">
      <p class="kh-demo__who">The API receives</p>
<pre class="kh-demo__code">Authorization: Bearer
<span class="kh-value">tok-fake-not-a-real-secret</span></pre>
    </div>
    <div class="kh-demo__pane kh-demo__pane--agent">
      <p class="kh-demo__who">The agent reads back</p>
<pre class="kh-demo__code">api says: hello, caller with
header Authorization: Bearer
<span class="kh-hidden">[hidden:vault://my-project/dev/API_TOKEN]</span></pre>
    </div>
  </div>
  <p class="kh-demo__note">A recorded run with a fake token, checked on every CI build: <a href="tutorials/quickstart/">see the quickstart</a>.</p>
</div>

</div>

## Why not just give the agent the key

Agents need API keys, cloud credentials and tokens to do real work. Pasted into the chat, a `.env` file or a command the model writes, a credential ends up in the transcript, in logs and in places you will forget. The other OpenBao and Vault MCP servers we found have a tool that reads a secret for the model. Keyholder has none: the model only ever holds names.

## A run, start to finish

<ol class="kh-steps">
  <li><strong>You store the secret</strong><p>In OpenBao, one path per project. The OpenBao web UI or the <code>bao</code> CLI, outside the chat.</p></li>
  <li><strong>You issue a session</strong><p>A wrapped, single-use token with a TTL, given to <code>keyholder-mcp</code> through its environment.</p></li>
  <li><strong>The agent runs a command</strong><p>It sends <code>{{vault://…}}</code>. Keyholder resolves it at execution, runs an allowed command without a shell, masks the output.</p></li>
  <li><strong>Access ends</strong><p>At the TTL every call fails with a clear message. You renew outside the chat; the agent cannot.</p></li>
</ol>

## How it compares

<div class="kh-compare" markdown>

| | Keyholder MCP | [mcp-vault](https://github.com/Cloud-Temple/mcp-vault) | [openbao-mcp](https://github.com/Knuckles-Team/openbao-mcp) | [vault-mcp-server](https://github.com/hashicorp/vault-mcp-server) |
| --- | --- | --- | --- | --- |
| Backend | OpenBao | OpenBao, embedded | OpenBao | HashiCorp Vault |
| Model receives secret values | <span class="kh-yes">Never</span> | <span class="kh-no">Yes, documented</span> | <span class="kh-no">Has a read tool</span> | <span class="kh-no">Yes, documented</span> |
| Value injected at run time | <span class="kh-yes">Yes</span> | <span class="kh-unk">Not documented</span> | <span class="kh-unk">Not documented</span> | <span class="kh-unk">Not documented</span> |
| Output masked | <span class="kh-yes">Yes</span> | <span class="kh-unk">Not documented</span> | <span class="kh-unk">Not documented</span> | <span class="kh-unk">Not documented</span> |
| Cloud credentials per run | <span class="kh-yes">AWS, GCP; revoked after the run</span> | <span class="kh-unk">Not documented</span> | <span class="kh-unk">Not documented</span> | <span class="kh-unk">Not documented</span> |
| Temporary access | <span class="kh-yes">Wrapped token, TTL</span> | <span class="kh-yes">Wrapping, SSH CA, PKI</span> | <span class="kh-unk">Not documented</span> | <span class="kh-unk">Not documented</span> |
| What the agent can do | Run allowed commands | 39 tools: secrets, SSH, PKI, policies | Vault operations | Mounts, KV, PKI |

</div>

From each project's README on 2026-10-14. They manage the vault for the agent; Keyholder only lets the agent use it. Details and sources: [How it compares](comparison.md).

## What it protects, and what it does not

<div class="kh-two" markdown>

<div markdown>

**Protects**

- Values stay out of the chat, tool results, errors and logs.
- A leaked session token dies with its TTL; a wrapped token works once.
- Only commands you name run, and never through a shell.
- References outside the project scope are refused.

</div>

<div markdown>

**Does not**

- Hide the value from the command it runs: that command needs it.
- Catch an encoded value: an allowed interpreter can escape the mask, as [the masking tutorial](tutorials/masking.md) shows.
- Stop a hostile agent running as your own OS user.

</div>

</div>

The full list is in the [security model](security.md).

## Start in five minutes

```bash
bao server -dev -dev-root-token-id=root-dev        # in-memory, for a first try
export BAO_ADDR=http://127.0.0.1:8200 BAO_TOKEN=root-dev
bao kv put -mount=secret my-project/dev API_TOKEN=your-api-token
bao policy write my-project-read deploy/policy-project-read.hcl
bao token create -policy=my-project-read -ttl=8h -explicit-max-ttl=8h -wrap-ttl=5m -field=wrapping_token
```

Then add `keyholder-mcp` to [Claude Code, Cursor or Codex](clients.md) with that wrapped token. Every step, explained: [Getting started](getting-started.md). Every behaviour, with real output: [Tutorials](tutorials/index.md).

Version 0.4.0, MIT licensed. [Source, issues and releases on GitHub](https://github.com/Mpaape/keyholder-mcp). Cloud CLIs get a credential made for each run: [cloud credentials](tutorials/cloud.md). For real secrets, run a [persistent vault](server.md). Next on the [roadmap](spec.md#11-roadmap): access requests from the agent, SSH.

</div>
