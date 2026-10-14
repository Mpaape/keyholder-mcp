// keyholder-mcp é um servidor MCP (stdio) que deixa agentes de IA usarem
// credenciais do OpenBao sem ver os valores.
//
// Configuração por variáveis de ambiente:
//
//	BAO_ADDR              endereço do OpenBao (padrão do cliente: https://127.0.0.1:8200)
//	BAO_WRAPPING_TOKEN    token embrulhado, de uso único (preferido)
//	BAO_TOKEN             token comum (aceito para a prova de conceito)
//	KEYHOLDER_KV_MOUNT    mount do KV v2 (padrão: secret)
//	KEYHOLDER_SCOPE       prefixo de caminho do projeto, por exemplo meu-projeto
//	KEYHOLDER_ALLOW       comandos permitidos, separados por vírgula, por exemplo curl,aws
//	KEYHOLDER_TIMEOUT     tempo máximo por execução (padrão: 5m)
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"keyholder-mcp/internal/bao"
	"keyholder-mcp/internal/runner"
)

var version = "0.1.0-dev"

type config struct {
	mount   string
	scope   string
	allow   map[string]bool
	timeout time.Duration
}

func loadConfig() (config, error) {
	c := config{
		mount:   envOr("KEYHOLDER_KV_MOUNT", "secret"),
		scope:   strings.Trim(os.Getenv("KEYHOLDER_SCOPE"), "/"),
		allow:   map[string]bool{},
		timeout: runner.DefaultTimeout,
	}
	for _, name := range strings.Split(os.Getenv("KEYHOLDER_ALLOW"), ",") {
		if name = strings.TrimSpace(name); name != "" {
			c.allow[name] = true
		}
	}
	if v := os.Getenv("KEYHOLDER_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return c, fmt.Errorf("KEYHOLDER_TIMEOUT inválido: %q", v)
		}
		c.timeout = d
	}
	return c, nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type listOutput struct {
	Scope   string      `json:"scope"`
	Entries []bao.Entry `json:"entries"`
	Note    string      `json:"note"`
}

var errNoSession = errors.New("sem sessão com o cofre; o humano precisa renovar o acesso fora do chat")

func main() {
	showVersion := flag.Bool("version", false, "mostra a versão e sai")
	flag.Parse()
	if *showVersion {
		fmt.Println("keyholder-mcp", version)
		return
	}

	// stdout é do protocolo MCP; logs e auditoria vão para stderr.
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.LUTC)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Sem sessão o servidor sobe mesmo assim: keyholder_status explica o motivo
	// e as outras tools falham de forma previsível (FL-07).
	store, openErr := bao.Open(ctx, cfg.mount, cfg.scope)
	if openErr != nil {
		log.Printf("sessão não aberta: %v", openErr)
	}
	if cfg.scope == "" {
		log.Printf("aviso: KEYHOLDER_SCOPE vazio; todas as referências do mount %q ficam acessíveis", cfg.mount)
	}

	run := &runner.Runner{
		Allow:   cfg.allow,
		Scope:   cfg.scope,
		Timeout: cfg.timeout,
	}
	if store != nil {
		run.Resolve = store.Resolve
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "keyholder-mcp", Version: version}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "keyholder_status",
		Description: "Estado do cofre e da sessão: selado ou não, TTL restante do token e políticas. Nunca devolve tokens nem valores.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, bao.Status, error) {
		if store == nil {
			return nil, bao.Status{Mount: cfg.mount, Scope: cfg.scope, SessionError: openErr.Error()}, nil
		}
		st, err := store.Status(ctx)
		return nil, st, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "keyholder_listar",
		Description: "Lista as referências vault:// disponíveis no escopo do projeto. Devolve só nomes, nunca valores.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, listOutput, error) {
		if store == nil {
			return nil, listOutput{}, errNoSession
		}
		entries, err := store.List(ctx)
		if err != nil {
			return nil, listOutput{}, err
		}
		if entries == nil {
			entries = []bao.Entry{}
		}
		return nil, listOutput{
			Scope:   cfg.scope,
			Entries: entries,
			Note:    "Use {{vault://caminho/CHAVE}} em args ou env de keyholder_executar.",
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "keyholder_executar",
		Description: "Executa um comando permitido, sem shell, trocando {{vault://caminho/CHAVE}} pelo valor só no momento da execução. " +
			"Exemplo: command=curl, args=[\"-H\", \"Authorization: Bearer {{vault://meu-projeto/dev/API_TOKEN}}\", \"https://api.exemplo.com/health\"]. " +
			"A saída volta com os valores mascarados. O mascaramento só pega o valor exato: não codifique nem imprima segredos de propósito.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in runner.Request) (*mcp.CallToolResult, runner.Result, error) {
		if store == nil {
			return nil, runner.Result{}, errNoSession
		}
		start := time.Now()
		res, err := run.Run(ctx, in)
		// Auditoria: comando, referências e resultado. Nunca valores nem argumentos.
		status := "ok"
		if err != nil {
			status = "erro: " + err.Error()
		}
		log.Printf("audit cmd=%q refs=%v exit=%d dur=%s status=%s", in.Command, res.RefsUsed, res.ExitCode, time.Since(start).Round(time.Millisecond), status)
		return nil, res, err
	})

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("servidor MCP: %v", err)
	}
}
