---
name: solid
version: 1
paths: ["cmd/**", "internal/**"]
---
SOLID principles applied to this project's Go code. In Go, "class" becomes package and type, and "inheritance" becomes composition and interfaces.

## SOLID-S One responsibility per package
Each package has one reason to change: `ref` parses references, `mask` hides values, `runner` executes, `bao` talks to OpenBao, `cmd/keyholder-mcp` wires everything to MCP. Vault logic in the runner, or execution logic in `bao`, is a finding. Functions with more than one clear responsibility should be split.

## SOLID-O Open for extension, closed for modification
New secret backends or adapters (AWS, GCP, SSH) come in as new `runner.Resolver` implementations or a new package, without changing the core flow of `Runner.Run`. A `switch` on backend type inside the runner is a finding.

## SOLID-L Substitution without surprise
Every `runner.Resolver` honors the same contract: it returns the value or an error without the value, honors the `context.Context`, and has no side effect the agent can see. A test fake and `bao.Store` must be interchangeable.

## SOLID-I Small interfaces, defined by the consumer
Interfaces and function types are declared in the package that uses them (like `runner.Resolver`), with the smallest set of methods needed. No interfaces "just in case" and no large interfaces exported by the producer.

## SOLID-D Depend on abstractions
severity: error
`internal/runner`, `internal/ref` and `internal/mask` do not import `internal/bao`, the OpenBao SDK or the MCP SDK. Concrete dependencies are wired only in `cmd/`. An import that inverts this direction blocks the merge.
