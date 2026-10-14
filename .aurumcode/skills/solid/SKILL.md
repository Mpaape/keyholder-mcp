---
name: solid
version: 1
paths: ["cmd/**", "internal/**"]
---
Princípios SOLID aplicados ao Go deste projeto. Em Go, "classe" vira pacote e tipo, e "herança" vira composição e interface.

## SOLID-S Responsabilidade única por pacote
Cada pacote tem uma razão para mudar: `ref` interpreta referências, `mask` oculta valores, `runner` executa, `bao` fala com o OpenBao, `cmd/keyholder-mcp` liga tudo ao MCP. Lógica de cofre no runner, ou de execução no pacote `bao`, é achado. Funções com mais de uma responsabilidade clara devem ser divididas.

## SOLID-O Aberto para extensão, fechado para modificação
Novos backends de segredo ou adaptadores (AWS, GCP, SSH) entram como novas implementações de `runner.Resolver` ou de um novo pacote, sem alterar o fluxo central de `Runner.Run`. Um `switch` por tipo de backend dentro do runner é achado.

## SOLID-L Substituição sem surpresa
Toda implementação de `runner.Resolver` respeita o mesmo contrato: devolve o valor ou um erro sem o valor, honra o `context.Context` e não tem efeito colateral visível ao agente. Um fake de teste e o `bao.Store` precisam ser intercambiáveis.

## SOLID-I Interfaces pequenas, definidas por quem consome
Interfaces e tipos de função são declarados no pacote que os usa (como `runner.Resolver`), com o menor conjunto de métodos necessário. Não criar interfaces "por precaução" nem exportar interfaces grandes do pacote produtor.

## SOLID-D Dependência de abstrações
severity: error
`internal/runner`, `internal/ref` e `internal/mask` não importam `internal/bao`, o SDK do OpenBao nem o SDK do MCP. As dependências concretas são ligadas só em `cmd/keyholder-mcp`. Um import que inverta essa direção é bloqueante.
