# Tutorial: mascaramento e seu limite

## Objetivo

Ao final você vai ver o mascaramento funcionar num valor que a API devolve, e ver exatamente onde ele deixa de funcionar.

Cada comando e cada saída abaixo vêm de uma execução real, gravada em `demo/tutorials/masking/expected/` e conferida por `run.sh --check`. As saídas são do programa, em inglês.

## Como rodar

```bash
bash demo/tutorials/masking/run.sh all
bash demo/tutorials/masking/run.sh --check
```

Script: [`demo/tutorials/masking/run.sh`](https://github.com/Mpaape/keyholder-mcp/blob/main/demo/tutorials/masking/run.sh).

## 1. Um valor refletido é mascarado

A API coloca o cabeçalho recebido na resposta. O agente lê a referência, não o valor.

```text
--8<-- "demo/tutorials/masking/expected/01-reflected.txt"
```

## 2. LIMITE: codificar escapa do mascaramento

`python3` está liberado aqui **só** para mostrar o limite. O interpretador codifica o valor em base64 e o agente o lê, sem máscara. O mascaramento só casa o valor exato; por isso a lista de comandos nunca deve incluir interpretadores ou shells (`sh`, `bash`, `python3`, `node`, `env`).

```text
--8<-- "demo/tutorials/masking/expected/02-limit-encoded.txt"
```
