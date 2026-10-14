# Windows e WSL

O `keyholder-mcp` é um binário Go único para Windows, Linux e macOS, e o CI roda os testes nos três. No Windows, a questão é onde o OpenBao roda e como o lado do agente o alcança.

## Topologia recomendada

Rode o **OpenBao dentro do WSL** (ou num contêiner) e deixe o Windows acessá-lo por `localhost`.

| Topologia | Como o outro lado conecta | Risco |
| --- | --- | --- |
| **A. OpenBao no WSL ou num contêiner** (recomendada) | O Windows usa `localhost`, que a [Microsoft documenta](https://learn.microsoft.com/en-us/windows/wsl/networking) para apps do WSL no modo NAT padrão | O cofre para quando o WSL desliga |
| B. OpenBao no Windows, WSL em NAT | O WSL usa o IP do host Windows | O OpenBao precisa escutar além de `127.0.0.1`, o que o expõe à rede local |
| C. OpenBao no Windows, WSL em mirrored | `localhost` nos dois sentidos, segundo a Microsoft | WSL → Windows por `127.0.0.1` tem falha de handshake TCP reportada e reproduzida no modo mirrored ([microsoft/WSL#40343](https://github.com/microsoft/WSL/issues/40343)), mesmo com `hostAddressLoopback=true` |

Em qualquer topologia, o `keyholder-mcp` roda ao lado do cliente MCP (Windows ou WSL, onde o agente estiver) e só precisa de `BAO_ADDR` e do token embrulhado.

## Instalar o OpenBao

Pela [página de instalação do OpenBao](https://openbao.org/docs/install/): Homebrew no macOS, pacotes para as principais distribuições Linux, imagens de contêiner (`ghcr.io/openbao/openbao`) e um zip com o binário `bao` (`bao.exe` no Windows). A mesma página recomenda desativar ou criptografar o swap, para segredos não irem para o disco; no Windows:

```powershell
fsutil behavior query encryptpagingfile
```

## Servidor dev e uso real

O `bao server -dev` serve para experimentar: dados em memória, `127.0.0.1:8200`, sem TLS, tudo perdido ao reiniciar ([documentação do OpenBao](https://openbao.org/docs/concepts/dev-server/)). Uso real exige armazenamento persistente, TLS e uma política de unseal; o servidor inicia **selado** a cada reinício ([seal](https://openbao.org/docs/concepts/seal/)).

## Compilar no Windows

```powershell
go mod tidy
go test ./...
go build -o keyholder-mcp.exe ./cmd/keyholder-mcp
```

Os testes do runner usam uma cópia do binário de teste como processo filho, então rodam no Windows sem shell.
