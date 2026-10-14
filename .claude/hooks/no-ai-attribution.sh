#!/usr/bin/env bash
# Claude Code PreToolUse hook (Bash). Blocks a command that would publish AI
# attribution: git commit messages, gh pr/issue bodies, gh api payloads.
# Exit code 2 blocks the tool call and shows stderr to Claude.
set -uo pipefail
root="${CLAUDE_PROJECT_DIR:-$(pwd)}"
patterns="$root/scripts/attribution/patterns.txt"
[ -f "$patterns" ] || exit 0
input="$(cat)"
cmd="$(printf '%s' "$input" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("tool_input",{}).get("command",""))' 2>/dev/null)"
case "$cmd" in
  *"git commit"*|*"git tag"*|*"gh pr"*|*"gh issue"*|*"gh api"*|*"gh release"*) ;;
  *) exit 0 ;;
esac
# The command text plus any file it passes as a message or body.
text="$cmd"
for f in $(printf '%s' "$cmd" | grep -oE -- '(-F|--file|--body-file|--input|--notes-file)[= ]+[^ ;&|]+' | sed -E 's/^[^ =]+[= ]+//'); do
  [ -f "$f" ] && text="$text"$'\n'"$(cat "$f")"
done
if hits="$(printf '%s\n' "$text" | grep -Ei -f <(grep -v '^#' "$patterns"))"; then
  echo "Blocked: AI attribution is not allowed (scripts/attribution/patterns.txt). Remove it and retry:" >&2
  printf '  %s\n' "$hits" >&2
  exit 2
fi
exit 0
