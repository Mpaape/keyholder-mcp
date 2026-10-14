# Windows and WSL

`keyholder-mcp` is a single Go binary for Windows, Linux and macOS, and CI runs the tests on all three. The question on Windows is where OpenBao runs and how the agent's side reaches it.

## Recommended topology

Run **OpenBao inside WSL** (or in a container), and let Windows reach it through `localhost`.

| Topology | How the other side connects | Risk |
| --- | --- | --- |
| **A. OpenBao in WSL or a container** (recommended) | Windows uses `localhost`, which [Microsoft documents](https://learn.microsoft.com/en-us/windows/wsl/networking) for apps running in WSL in the default NAT mode | The vault stops when WSL shuts down |
| B. OpenBao on Windows, WSL in NAT mode | WSL uses the Windows host IP | OpenBao must listen beyond `127.0.0.1`, which exposes it to the LAN |
| C. OpenBao on Windows, WSL in mirrored mode | `localhost` in both directions, per Microsoft | WSL → Windows over `127.0.0.1` has a reported, reproduced TCP handshake failure in mirrored mode ([microsoft/WSL#40343](https://github.com/microsoft/WSL/issues/40343)), even with `hostAddressLoopback=true` |

Whatever the topology, `keyholder-mcp` runs next to the MCP client (Windows or WSL, wherever the agent runs) and only needs `BAO_ADDR` and the wrapped token.

## Install OpenBao

Per the [OpenBao install page](https://openbao.org/docs/install/): Homebrew on macOS, packages for the main Linux distributions, container images (`ghcr.io/openbao/openbao`), and a zip with the `bao` binary (`bao.exe` on Windows). The same page recommends disabling or encrypting swap, so secrets are not paged to disk; on Windows:

```powershell
fsutil behavior query encryptpagingfile
```

## Dev server versus real use

`bao server -dev` is for trying things out: data in memory, `127.0.0.1:8200`, no TLS, lost on restart ([OpenBao docs](https://openbao.org/docs/concepts/dev-server/)). Real use needs persistent storage, TLS and an unseal policy; the server starts **sealed** after every restart ([seal concepts](https://openbao.org/docs/concepts/seal/)).

## Building on Windows

```powershell
go mod tidy
go test ./...
go build -o keyholder-mcp.exe ./cmd/keyholder-mcp
```

The runner tests use a copy of the test binary as the child process, so they run on Windows without a shell.
