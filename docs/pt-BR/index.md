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

<p class="kh-hero__title">Seu agente de IA usa a credencial. E nunca a vê.</p>

<p class="kh-hero__lede">O Keyholder MCP é um servidor <a href="https://modelcontextprotocol.io">MCP</a> sobre o <a href="https://openbao.org">OpenBao</a>. O agente escreve uma referência; o Keyholder coloca o valor real no comando só na hora de executar e devolve a saída com o valor mascarado. O acesso acaba sozinho.</p>

[Começar](getting-started.md){ .md-button .md-button--primary } [Baixar](https://github.com/Mpaape/keyholder-mcp/releases){ .md-button }

</div>

<div class="kh-demo" aria-label="Uma requisição vista pela API e pelo agente">
  <div class="kh-demo__pane">
    <p class="kh-demo__who">O agente pede ao keyholder_run</p>
<pre class="kh-demo__code">curl -sS \
  -H "Authorization: Bearer <span class="kh-ref">{{vault://my-project/dev/API_TOKEN}}</span>" \
  <span class="kh-ref">{{vault://my-project/dev/API_URL}}</span></pre>
  </div>
  <div class="kh-demo__fork" aria-hidden="true"></div>
  <div class="kh-demo__split">
    <div class="kh-demo__pane kh-demo__pane--api">
      <p class="kh-demo__who">A API recebe</p>
<pre class="kh-demo__code">Authorization: Bearer
<span class="kh-value">tok-fake-not-a-real-secret</span></pre>
    </div>
    <div class="kh-demo__pane kh-demo__pane--agent">
      <p class="kh-demo__who">O agente lê de volta</p>
<pre class="kh-demo__code">api says: hello, caller with
header Authorization: Bearer
<span class="kh-hidden">[hidden:vault://my-project/dev/API_TOKEN]</span></pre>
    </div>
  </div>
  <p class="kh-demo__note">Uma execução gravada com um token falso, conferida a cada build do CI: <a href="tutorials/quickstart/">veja o início rápido</a>.</p>
</div>

</div>

## Por que não dar a chave ao agente

Agentes precisam de chaves de API, credenciais de nuvem e tokens para fazer trabalho real. Colada no chat, num `.env` ou num comando escrito pelo modelo, a credencial vai parar na conversa, em logs e em lugares que você vai esquecer. Os outros servidores MCP de OpenBao e Vault que encontramos têm uma tool que lê o segredo para o modelo. O Keyholder não tem nenhuma: o modelo só segura nomes.

## Uma execução, do começo ao fim

<ol class="kh-steps">
  <li><strong>Você guarda o segredo</strong><p>No OpenBao, um caminho por projeto. Pela interface web do OpenBao ou pela CLI <code>bao</code>, fora do chat.</p></li>
  <li><strong>Você emite uma sessão</strong><p>Um token embrulhado, de uso único, com TTL, entregue ao <code>keyholder-mcp</code> pelo ambiente dele.</p></li>
  <li><strong>O agente roda um comando</strong><p>Ele envia <code>{{vault://…}}</code>. O Keyholder resolve na execução, roda um comando permitido sem shell e mascara a saída.</p></li>
  <li><strong>O acesso acaba</strong><p>No fim do TTL toda chamada falha com uma mensagem clara. Você renova fora do chat; o agente não consegue.</p></li>
</ol>

## Comparação

<div class="kh-compare" markdown>

| | Keyholder MCP | [mcp-vault](https://github.com/Cloud-Temple/mcp-vault) | [openbao-mcp](https://github.com/Knuckles-Team/openbao-mcp) | [vault-mcp-server](https://github.com/hashicorp/vault-mcp-server) |
| --- | --- | --- | --- | --- |
| Cofre | OpenBao | OpenBao, embutido | OpenBao | HashiCorp Vault |
| O modelo recebe valores | <span class="kh-yes">Nunca</span> | <span class="kh-no">Sim, documentado</span> | <span class="kh-no">Tem tool de leitura</span> | <span class="kh-no">Sim, documentado</span> |
| Valor injetado na execução | <span class="kh-yes">Sim</span> | <span class="kh-unk">Não documentado</span> | <span class="kh-unk">Não documentado</span> | <span class="kh-unk">Não documentado</span> |
| Saída mascarada | <span class="kh-yes">Sim</span> | <span class="kh-unk">Não documentado</span> | <span class="kh-unk">Não documentado</span> | <span class="kh-unk">Não documentado</span> |
| Acesso temporário | <span class="kh-yes">Token embrulhado, TTL</span> | <span class="kh-yes">Wrapping, CA de SSH, PKI</span> | <span class="kh-unk">Não documentado</span> | <span class="kh-unk">Não documentado</span> |
| O que o agente faz | Roda comandos permitidos | 39 tools: segredos, SSH, PKI, políticas | Operações do cofre | Mounts, KV, PKI |

</div>

Pelo README de cada projeto em 2026-10-14. Eles administram o cofre pelo agente; o Keyholder só deixa o agente usá-lo. Detalhes e fontes: [Comparação](comparison.md).

## O que protege, e o que não

<div class="kh-two" markdown>

<div markdown>

**Protege**

- Os valores ficam fora do chat, dos resultados das tools, dos erros e dos logs.
- Um token de sessão vazado morre com o TTL; um token embrulhado funciona uma vez.
- Só rodam os comandos que você nomeia, e nunca por um shell.
- Referências fora do escopo do projeto são recusadas.

</div>

<div markdown>

**Não protege**

- O comando executado recebe o valor: ele precisa dele.
- Um valor codificado escapa: um interpretador liberado contorna o mascaramento, como mostra [o tutorial de mascaramento](tutorials/masking.md).
- Um agente hostil rodando com o seu usuário do sistema.

</div>

</div>

A lista completa está no [modelo de segurança](security.md).

## Comece em cinco minutos

```bash
bao server -dev -dev-root-token-id=root-dev        # em memória, para experimentar
export BAO_ADDR=http://127.0.0.1:8200 BAO_TOKEN=root-dev
bao kv put -mount=secret my-project/dev API_TOKEN=seu-token-de-api
bao policy write my-project-read deploy/policy-project-read.hcl
bao token create -policy=my-project-read -ttl=8h -explicit-max-ttl=8h -wrap-ttl=5m -field=wrapping_token
```

Depois cadastre o `keyholder-mcp` no [Claude Code, Cursor ou Codex](clients.md) com esse token embrulhado. Cada passo explicado: [Começando](getting-started.md). Cada comportamento, com saída real: [Tutoriais](tutorials/index.md).

Versão 0.1.0, licença MIT. [Código, issues e releases no GitHub](https://github.com/Mpaape/keyholder-mcp). Próximos passos no [roteiro](spec.md#11-roteiro): credenciais dinâmicas de AWS e GCP, pedidos de acesso pelo agente, SSH.

</div>
