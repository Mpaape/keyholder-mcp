# Code review

Every pull request is reviewed by [AurumCode](https://github.com/Mpaape/AurumCode) v2.0.0 with the project's own rules. The `aurumcode/policy-gate` check fails the PR when there is an `error` finding.

## How it is wired

| File | Role |
| --- | --- |
| `.github/workflows/aurumcode.yml` | Calls the reusable workflow `Mpaape/AurumCode/.github/workflows/review.yml@v2.0.0` on each PR |
| `.aurumcode/config.yml` | Language, publication, gate, scanners, deliberation and exceptions |
| `.aurumcode/prompt.md` | Project context sent to the model |
| `.aurumcode/skills/<name>/SKILL.md` | Project rules; each `## ` section becomes a citable rule |

Configuration and skills are read from the PR's **base branch**: a new rule applies only after it is merged into `main`.

| Job | Does |
| --- | --- |
| `review` | Model review, deterministic security pass, gitleaks and Semgrep |
| `upload-sarif` | Sends the SARIF to the Security tab (code scanning); skipped on fork PRs |
| `provider` | Fails closed: without `LLM_API_KEY` and `LLM_BASE_URL`, marks `aurumcode/policy-gate` as failed |

## Secrets and variables

The same names as the AurumCode repository. Settings → Secrets and variables → Actions.

| Kind | Name | Required | Value with the Anthropic API |
| --- | --- | --- | --- |
| Secret | `LLM_API_KEY` | Yes | Your Claude API key |
| Secret | `LLM_BASE_URL` | Yes | `https://api.anthropic.com/v1` |
| Variable | `LLM_PROVIDER` | No | `anthropic` |
| Variable | `LLM_MODEL` | Depends on the provider | Model id, for example `claude-haiku-5-5` |
| Secret | `LLM_FALLBACK_1_API_KEY`, `LLM_FALLBACK_1_BASE_URL` | No | Fallback provider |
| Variable | `LLM_FALLBACK_1_PROVIDER`, `LLM_FALLBACK_1_MODEL` | No | Fallback provider |

With the [GitHub CLI](https://cli.github.com/); `gh secret set` without `--body` prompts for the value without echoing it:

```bash
gh secret set LLM_API_KEY -R Mpaape/keyholder-mcp
gh secret set LLM_BASE_URL -R Mpaape/keyholder-mcp --body "https://api.anthropic.com/v1"
gh variable set LLM_PROVIDER -R Mpaape/keyholder-mcp --body "anthropic"
gh variable set LLM_MODEL -R Mpaape/keyholder-mcp --body "claude-haiku-5-5"
```

Provider profiles and URLs: [AurumCode providers](https://github.com/Mpaape/AurumCode/blob/v2.0.0/docs/provedores.md). Dependabot PRs only get *Dependabot secrets*; to review them, add `LLM_API_KEY` and `LLM_BASE_URL` there too.

## Gate

| Setting | Effect |
| --- | --- |
| `gate.fail_on: [error]` | An `error` finding from the model, a skill or a scanner fails the check |
| `gate.inconclusive: block` | A review that did not finish (provider down, scanner failed) never passes |
| `gitleaks`, `required: true` | Secrets in any commit of the PR, not only in the final tree |
| `semgrep` | SAST; fails only on lines the PR added |
| `govet` | Off until `go.sum` is in the repository |

To make the check block merges, after the first PR: Settings → Branches → rule for `main` → *Require status checks to pass before merging* → `aurumcode/policy-gate`.

## Project rules

Citable id: `<skill>#<slug of the heading>`, for example `security#sec-001`. Headings carry only the id and the title opens the body: AurumCode v2.0.0 puts every rule id in the prompt under a fixed 512-token budget and refuses to run when it overflows, so long ids would stop the review. Rules marked "Yes" have `severity: error` and block the merge; the others are warnings.

### `security` — paths ["**"]

| Rule | Blocks |
| --- | --- |
| SEC-001 A secret value never goes back to the agent | Yes |
| SEC-002 A secret never goes to a log, an error or a file | Yes |
| SEC-003 Execution without a shell | Yes |
| SEC-004 Command allow list | Yes |
| SEC-005 The vault token stays out of the child process | Yes |
| SEC-006 Project scope | Yes |
| SEC-007 Reference validation | Yes |
| SEC-008 Masking covers all output | Yes |
| SEC-009 Token in memory only | Yes |
| SEC-010 TLS and vault address | Yes |
| SEC-011 Bounded output and time | No |
| SEC-012 New dependencies are justified | No |
| SEC-013 Public repository: nothing internal | Yes |
| SEC-014 Examples and tests use fake values only | Yes |

### `review` — paths ["cmd/**", "internal/**"]

| Rule | Blocks |
| --- | --- |
| REV-001 Errors are handled and carry context | Yes |
| REV-002 New behavior comes with a test | Yes |
| REV-003 Context is propagated | No |
| REV-004 stdout belongs to the MCP protocol | Yes |
| REV-005 No panic on a request path | Yes |
| REV-006 Windows, Linux and macOS | No |
| REV-007 Messages to the agent are actionable | No |
| REV-008 Formatted and vetted code | No |
| REV-009 Docs follow the change | No |

### `solid` — paths ["cmd/**", "internal/**"]

| Rule | Blocks |
| --- | --- |
| SOLID-S One responsibility per package | No |
| SOLID-O Open for extension, closed for modification | No |
| SOLID-L Substitution without surprise | No |
| SOLID-I Small interfaces, defined by the consumer | No |
| SOLID-D Depend on abstractions | Yes |

To change a rule, edit its section in the `SKILL.md`. To add one, add a `## ` section with an id and a title; `severity: error` on the first line of the body makes it blocking.

## Approved exceptions

A reviewed false positive goes in `exceptions` in `.aurumcode/config.yml`, with the exact rule and path, owner, reason and expiry. It stops applying on its own at the date.

| Rule | File | Reason | Expires |
| --- | --- | --- | --- |
| `analysis/hardcoded-secret` | `internal/bao/bao.go` | `EnvWrappingToken` and `EnvToken` hold environment variable names, not secrets | 2027-04-13 |

## Running it locally

With the AurumCode CLI (Go 1.27 or its Docker image), at the repository root:

```bash
aurumcode review --base origin/main --seguranca
```

Without `LLM_API_KEY` and `LLM_BASE_URL` only the deterministic analysis runs; without `gitleaks` and `semgrep` on the `PATH` those scanners are reported as inconclusive. See [AurumCode getting started](https://github.com/Mpaape/AurumCode/blob/v2.0.0/docs/getting-started.md).
