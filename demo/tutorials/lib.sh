# Shared helpers for the executable tutorials. Sourced by each run.sh.
#
# Requirements on the host: bash, go (or prebuilt binaries in KH_BIN_DIR),
# bao (OpenBao CLI), curl and python3 (openssl for the server tutorial).
# Nothing touches a real vault: every run starts its own OpenBao on
# 127.0.0.1:18200 (a dev server in memory, or the server tutorial's own server
# with its data in a temporary directory), with fake secrets, and stops it.
set -Eeuo pipefail

TUT_DIR="$(cd "$(dirname "${BASH_SOURCE[1]}")" && pwd)"
ROOT="$(cd "$TUT_DIR/../../.." && pwd)"
cd "$ROOT"   # every path in the records is relative to the repository root
OUT="$TUT_DIR/out"
EXPECTED="$TUT_DIR/expected"
WORK="$(mktemp -d)"
export BAO_ADDR="http://127.0.0.1:18200"
# Everything is local: no proxy, so the records do not depend on the host.
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY no_proxy NO_PROXY all_proxy ALL_PROXY
# No cloud credentials or settings from the host: the cloud tutorial gives
# the commands only what Keyholder injects.
for v in $(compgen -e | grep -E '^(AWS_|CLOUDSDK_|GOOGLE_)' || true); do unset "$v"; done
ROOT_TOKEN="root-dev"   # dev server root token: fake, local, in memory
PIDS=()
VAULT_ARGS=()

cleanup() {
  for p in "${PIDS[@]:-}"; do [ -n "$p" ] && kill "$p" 2>/dev/null || true; done
  rm -rf "$WORK"
}
trap cleanup EXIT

# --check compares out/ with expected/ without running anything.
check() {
  if diff -ru "$EXPECTED" "$OUT"; then
    echo "RESULT: out/ matches expected/"
  else
    echo "RESULT: out/ differs from expected/"; return 1
  fi
}

binaries() {
  if [ -n "${KH_BIN_DIR:-}" ]; then
    BIN="$KH_BIN_DIR"
  else
    BIN="$WORK/bin"; mkdir -p "$BIN"
    (cd "$ROOT" && go build -o "$BIN/keyholder-mcp" ./cmd/keyholder-mcp && go build -o "$BIN/khcall" ./cmd/khcall)
  fi
  export PATH="$BIN:$PATH"
}

# start_vault starts a fresh dev server and waits until it answers.
# Extra arguments go to the server, for example -dev-plugin-dir.
start_vault() {
  bao server -dev -dev-root-token-id="$ROOT_TOKEN" -dev-listen-address=127.0.0.1:18200 "${VAULT_ARGS[@]}" >"$WORK/bao.log" 2>&1 &
  PIDS+=($!)
  for _ in $(seq 1 50); do
    BAO_TOKEN="$ROOT_TOKEN" bao status >/dev/null 2>&1 && return 0
    sleep 0.2
  done
  echo "OpenBao dev server did not start"; cat "$WORK/bao.log"; return 1
}

# admin runs a bao command as the dev root (the human, outside the chat).
admin() { BAO_TOKEN="$ROOT_TOKEN" bao "$@"; }

# wrapped_token <policy> <ttl>: prints a wrapped, single-use token.
wrapped_token() {
  admin token create -policy="$1" -ttl="$2" -explicit-max-ttl="$2" -wrap-ttl=5m -field=wrapping_token
}

# echo_server starts a local HTTP server that answers with a fixed text and
# records the SHA-256 of the Authorization header it received.
echo_server() {
  python3 - "$WORK/seen" >"$WORK/http.log" 2>&1 <<'PY' &
import hashlib, http.server, sys
seen = sys.argv[1]
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        auth = self.headers.get("Authorization", "")
        open(seen, "w").write(hashlib.sha256(auth.encode()).hexdigest())
        body = ("api says: hello, caller with header Authorization: " + auth + "\n").encode()
        self.send_response(200); self.send_header("Content-Length", str(len(body))); self.end_headers()
        self.wfile.write(body)
    def log_message(self, *a): pass
http.server.HTTPServer(("127.0.0.1", 18080), H).serve_forever()
PY
  PIDS+=($!)
  for _ in $(seq 1 50); do curl -s -o /dev/null http://127.0.0.1:18080/ 2>/dev/null && return 0; sleep 0.1; done
  echo "echo server did not start"; return 1
}

# server_saw <expected header>: did the HTTP server receive exactly this header?
server_saw() {
  local want; want="$(printf '%s' "$1" | sha256sum | cut -d' ' -f1)"
  if [ "$(cat "$WORK/seen" 2>/dev/null)" = "$want" ]; then echo "yes"; else echo "no"; fi
}

# normalize hides what changes between runs: timestamps, durations, TTLs.
normalize() {
  sed -E \
    -e 's/[0-9]{4}\/[0-9]{2}\/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}/<timestamp>/g' \
    -e 's/dur=[0-9.]+(ms|s|µs)/dur=<duration>/g' \
    -e 's/"token_ttl_seconds": [1-9][0-9]*/"token_ttl_seconds": <ttl>/g' \
    -e 's/Date: [A-Z][a-z]{2}, [0-9]{2} [A-Z][a-z]{2} [0-9]{4} [0-9:]{8} GMT/Date: <date>/g' \
    -e 's/curl\/[0-9.]+/curl\/<version>/g' \
    -e 's/Python\/[0-9.]+/Python\/<version>/g' \
    -e 's/"token_expire_time": "[^"]*"/"token_expire_time": "<time>"/g'\
    -e 's/[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:.]+(Z|[+-][0-9:]+)/<time>/g' \
    -e 's/vault-token-([A-Za-z0-9_-]+)-[0-9]{9,}-[A-Za-z0-9]+/vault-token-\1-<id>/g' \
    -e 's/"UserId": "[A-Za-z0-9]+"/"UserId": "<id>"/g' \
    -e 's#(pki/issue/[A-Za-z0-9_-]+/)[A-Za-z0-9]+#\1<id>#g'
}

# step <case> <title>: starts a case file in out/.
step() {
  CASE_FILE="$OUT/$1.txt"
  mkdir -p "$OUT"
  printf '# %s\n' "$2" >"$CASE_FILE"
}

# show <command...>: runs a command, records "$ command", its output and exit code.
show() {
  local rc=0 display="$*"
  display="${display/#admin /bao }"   # admin is the human running bao as the vault administrator
  printf '\n$ %s\n' "$display" >>"$CASE_FILE"
  "$@" >"$WORK/stdout" 2>"$WORK/stderr" || rc=$?
  normalize <"$WORK/stdout" >>"$CASE_FILE"
  if [ -s "$WORK/stderr" ]; then
    printf -- '--- stderr\n' >>"$CASE_FILE"
    normalize <"$WORK/stderr" >>"$CASE_FILE"
  fi
  printf 'exit_code=%s\n' "$rc" >>"$CASE_FILE"
}

# note <text>: records a line of explanation or a computed result.
note() { printf '\n%s\n' "$*" >>"$CASE_FILE"; }

# show_as <display> <command...>: like show, but records <display> as the
# command line. Used to hide a wrapped token behind a placeholder.
show_as() {
  local display="$1"; shift
  local rc=0
  printf '\n$ %s\n' "$display" >>"$CASE_FILE"
  "$@" >"$WORK/stdout" 2>"$WORK/stderr" || rc=$?
  normalize <"$WORK/stdout" >>"$CASE_FILE"
  if [ -s "$WORK/stderr" ]; then
    printf -- '--- stderr\n' >>"$CASE_FILE"
    normalize <"$WORK/stderr" >>"$CASE_FILE"
  fi
  printf 'exit_code=%s\n' "$rc" >>"$CASE_FILE"
}

# session <policy> <ttl> <khcall args...>: one MCP session with a fresh
# wrapped token. The record shows the token as <wrapped token>.
session() {
  local policy="$1" ttl="$2"; shift 2
  local w; w="$(wrapped_token "$policy" "$ttl")"
  show_as "BAO_WRAPPING_TOKEN=<wrapped token> khcall $(quote "$@")" env BAO_WRAPPING_TOKEN="$w" khcall "$@"
}

# quote prints arguments the way a reader would type them.
quote() {
  local out="" a
  for a in "$@"; do
    case "$a" in
      *[[:space:]\{\"\'\*]*) out+=" '$a'" ;;
      *) out+=" $a" ;;
    esac
  done
  printf '%s' "${out# }"
}

# main dispatches: run.sh all | <case> | --check
main() {
  case "${1:-all}" in
    --check) check ;;
    all) rm -rf "$OUT"; binaries; start_vault; setup; for c in "${CASES[@]}"; do "case_$c"; done; echo "RESULT: out/ written ($(ls "$OUT" | wc -l) cases)" ;;
    *) binaries; start_vault; setup; "case_$1" ;;
  esac
}
