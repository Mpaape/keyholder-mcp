# Keyholder-themed vault UI

The OpenBao web UI, served through a reverse proxy that gives it the Keyholder
look: black-and-white, line-based, the Keyholder mark in place of the OpenBao
logo. It is **cosmetic**. The proxy injects one same-origin stylesheet and
serves the logo under `/kh/`; it changes no behaviour and touches no secret.

```bash
docker compose -f deploy/ui-theme/compose.yml up
# open http://127.0.0.1:8200/ui/  (dev token: root-dev)
```

- `kh/theme.css` restyles the UI through OpenBao's own stable class names.
- `kh/logo.svg`, `kh/mark.svg` are the Keyholder mark; swap them to rebrand.
- `nginx.conf` proxies OpenBao and injects the stylesheet into the UI shell.
- It respects OpenBao's Content-Security-Policy (`style-src`, `img-src`,
  `font-src` all `'self'`): everything is served from the same origin.

For a persistent, TLS vault, run OpenBao from [`deploy/server.hcl`](../server.hcl)
([guide](../../docs/server.md)) and set this proxy's `upstream` to it, or
terminate TLS at the proxy.
