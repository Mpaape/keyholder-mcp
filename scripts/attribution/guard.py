#!/usr/bin/env python3
"""Keeps AI attribution out of the repository's GitHub content.

Run by .github/workflows/attribution.yml:

- issues, issue_comment, pull_request_target: removes attribution lines and
  footers from the body that triggered the event, then saves it back.
- pull_request_target: fails when a commit message in the PR has an AI
  trailer or footer (scripts/attribution/patterns.txt).
- workflow_dispatch: cleans every issue, pull request and comment body.

Bodies are data only: they are never executed or interpolated in a shell.
"""
import json
import os
import re
import sys
import urllib.request

API = os.environ.get("GITHUB_API_URL", "https://api.github.com")
REPO = os.environ["GITHUB_REPOSITORY"]
TOKEN = os.environ["GITHUB_TOKEN"]
HERE = os.path.dirname(os.path.abspath(__file__))

# Footer blocks some tools append, with the separator line before them.
FOOTER = re.compile(
    r"\n*(?:-{3,}\s*\n)?\s*(?:🤖\s*)?_?Generated (?:with|by) \[Claude Code\]\([^)]*\)_?[^\n]*\s*$",
    re.IGNORECASE,
)


def load_patterns():
    pats = []
    with open(os.path.join(HERE, "patterns.txt"), encoding="utf-8") as fh:
        for line in fh:
            line = line.rstrip("\n")
            if line and not line.startswith("#"):
                # POSIX class used by grep; Python spells it \s.
                pats.append(re.compile(line.replace("[[:space:]]", r"\s"), re.IGNORECASE))
    return pats


PATTERNS = load_patterns()


def api(method, path, payload=None):
    url = path if path.startswith("http") else f"{API}{path}"
    data = json.dumps(payload).encode() if payload is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Authorization", f"Bearer {TOKEN}")
    req.add_header("Accept", "application/vnd.github+json")
    req.add_header("X-GitHub-Api-Version", "2022-11-28")
    with urllib.request.urlopen(req) as resp:
        body = resp.read()
        link = resp.headers.get("Link", "")
    return (json.loads(body) if body else None), link


def paged(path):
    url = f"{API}{path}{'&' if '?' in path else '?'}per_page=100"
    while url:
        items, link = api("GET", url)
        yield from items
        nxt = re.search(r'<([^>]+)>;\s*rel="next"', link)
        url = nxt.group(1) if nxt else None


def clean(body):
    if not body:
        return body
    out = body
    while True:
        new = FOOTER.sub("", out)
        if new == out:
            break
        out = new
    lines = out.split("\n")
    kept = [l for l in lines if not any(p.search(l) for p in PATTERNS)]
    if out == body and len(kept) == len(lines):
        return body  # nothing to remove: leave the body byte for byte
    out = "\n".join(kept).rstrip()
    return out + "\n" if out else out


def fix(kind, url, body):
    new = clean(body)
    if new != body:
        api("PATCH", url, {"body": new})
        print(f"cleaned {kind}: {url}")
        return True
    return False


def check_commits(number):
    bad = []
    for c in paged(f"/repos/{REPO}/pulls/{number}/commits"):
        msg = c["commit"]["message"]
        hits = [l for l in msg.split("\n") if any(p.search(l) for p in PATTERNS)]
        if hits:
            bad.append((c["sha"][:7], hits))
    for sha, hits in bad:
        for h in hits:
            print(f"::error title=AI attribution in commit {sha}::{h.strip()}")
    return not bad


def main():
    event = os.environ["GITHUB_EVENT_NAME"]
    with open(os.environ["GITHUB_EVENT_PATH"], encoding="utf-8") as fh:
        ev = json.load(fh)
    ok = True
    if event == "issues":
        fix("issue", ev["issue"]["url"], ev["issue"]["body"])
    elif event == "issue_comment":
        fix("comment", ev["comment"]["url"], ev["comment"]["body"])
    elif event == "pull_request_target":
        pr = ev["pull_request"]
        fix("pull request", f"/repos/{REPO}/issues/{pr['number']}", pr["body"])
        ok = check_commits(pr["number"])
    elif event == "workflow_dispatch":
        for it in paged(f"/repos/{REPO}/issues?state=all"):
            fix("issue/pr", it["url"], it["body"])
        for cm in paged(f"/repos/{REPO}/issues/comments"):
            fix("comment", cm["url"], cm["body"])
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
