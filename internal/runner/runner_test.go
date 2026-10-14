package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"keyholder-mcp/internal/ref"
)

// O próprio binário de teste faz o papel do comando executado, o que funciona
// igual em Linux, macOS e Windows. Ver TestMain.
const helperEnv = "KH_TEST_HELPER"

// helperDir guarda uma CÓPIA do binário de teste com o nome kh-helper. Não é um
// hard link: no Windows um executável em uso não pode ser apagado, e o link
// aponta para o binário que está rodando, então a limpeza do diretório falharia.
var helperDir string

func TestMain(m *testing.M) {
	if mode := os.Getenv(helperEnv); mode != "" {
		switch mode {
		case "echo": // imprime os argumentos e a variável API_KEY
			fmt.Println(strings.Join(os.Args[1:], " "))
			fmt.Fprintln(os.Stderr, "API_KEY="+os.Getenv("API_KEY"))
		case "env": // imprime nomes das variáveis herdadas
			for _, kv := range os.Environ() {
				name, _, _ := strings.Cut(kv, "=")
				fmt.Println(name)
			}
		case "fail":
			os.Exit(3)
		case "big":
			fmt.Print(strings.Repeat("x", 1000))
		}
		os.Exit(0)
	}
	dir, err := prepareHelper()
	if err != nil {
		fmt.Fprintln(os.Stderr, "preparar kh-helper:", err)
		os.Exit(1)
	}
	helperDir = dir
	code := m.Run()
	_ = os.RemoveAll(dir) // melhor esforço: antivírus pode segurar o arquivo por um instante
	os.Exit(code)
}

// prepareHelper copia o binário de teste para um diretório temporário.
func prepareHelper() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", "kh-helper-")
	if err != nil {
		return "", err
	}
	name := "kh-helper"
	if filepath.Ext(exe) == ".exe" {
		name += ".exe"
	}
	src, err := os.Open(exe)
	if err != nil {
		return "", err
	}
	defer src.Close()
	dst, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return "", err
	}
	return dir, dst.Close()
}

func helper(t *testing.T, mode string, extraEnv ...string) *Runner {
	t.Helper()
	dir := helperDir
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	secrets := map[string]string{
		"vault://proj/dev/TOKEN": "tok-SUPER-secreto",
		"vault://proj/dev/KEY":   "key-123456",
		"vault://outro/dev/X":    "fora-do-escopo",
	}
	return &Runner{
		Allow: map[string]bool{"kh-helper": true},
		Scope: "proj",
		Resolve: func(_ context.Context, r ref.Ref) (string, error) {
			v, ok := secrets[r.String()]
			if !ok {
				return "", errors.New("não encontrado")
			}
			return v, nil
		},
		BaseEnv: append([]string{helperEnv + "=" + mode, "PATH=" + os.Getenv("PATH")}, extraEnv...),
	}
}

func TestRunInjectsAndMasks(t *testing.T) {
	r := helper(t, "echo")
	res, err := r.Run(context.Background(), Request{
		Command: "kh-helper",
		Args:    []string{"Bearer {{vault://proj/dev/TOKEN}}", "literal"},
		Env:     map[string]string{"API_KEY": "{{vault://proj/dev/KEY}}"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.Stdout+res.Stderr, "SUPER-secreto") || strings.Contains(res.Stdout+res.Stderr, "key-123456") {
		t.Fatalf("valor vazou: stdout=%q stderr=%q", res.Stdout, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "Bearer [oculto:vault://proj/dev/TOKEN] literal") {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	if !strings.Contains(res.Stderr, "API_KEY=[oculto:vault://proj/dev/KEY]") {
		t.Fatalf("stderr = %q", res.Stderr)
	}
	if strings.Join(res.RefsUsed, ",") != "vault://proj/dev/TOKEN,vault://proj/dev/KEY" {
		t.Fatalf("refs = %v", res.RefsUsed)
	}
}

func TestRunRejectsNotAllowed(t *testing.T) {
	r := helper(t, "echo")
	for _, cmd := range []string{"sh", "", "../kh-helper", "{{vault://proj/dev/TOKEN}}"} {
		if _, err := r.Run(context.Background(), Request{Command: cmd}); !errors.Is(err, ErrNotAllowed) {
			t.Errorf("Run(%q) err = %v, want ErrNotAllowed", cmd, err)
		}
	}
}

func TestRunRejectsOutOfScope(t *testing.T) {
	r := helper(t, "echo")
	_, err := r.Run(context.Background(), Request{Command: "kh-helper", Args: []string{"{{vault://outro/dev/X}}"}})
	if !errors.Is(err, ErrOutOfScope) {
		t.Fatalf("err = %v, want ErrOutOfScope", err)
	}
}

func TestRunStripsVaultEnv(t *testing.T) {
	r := helper(t, "env", "BAO_TOKEN=s.nao-pode-vazar", "VAULT_TOKEN=x", "KEYHOLDER_SCOPE=proj", "HOME_OK=1")
	res, err := r.Run(context.Background(), Request{Command: "kh-helper"})
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"BAO_TOKEN", "VAULT_TOKEN", "KEYHOLDER_SCOPE"} {
		if strings.Contains(res.Stdout, bad) {
			t.Errorf("%s chegou ao processo filho", bad)
		}
	}
	if !strings.Contains(res.Stdout, "HOME_OK") {
		t.Errorf("variável comum foi removida: %q", res.Stdout)
	}
	if _, err := r.Run(context.Background(), Request{Command: "kh-helper", Env: map[string]string{"BAO_TOKEN": "x"}}); err == nil {
		t.Error("esperava recusa de BAO_TOKEN em env")
	}
}

func TestRunExitCodeAndTruncation(t *testing.T) {
	res, err := helper(t, "fail").Run(context.Background(), Request{Command: "kh-helper"})
	if err != nil || res.ExitCode != 3 {
		t.Fatalf("exit = %d err = %v", res.ExitCode, err)
	}
	r := helper(t, "big")
	r.MaxOutput = 100
	res, err = r.Run(context.Background(), Request{Command: "kh-helper"})
	if err != nil || !res.Truncated || len(res.Stdout) != 100 {
		t.Fatalf("truncated=%v len=%d err=%v", res.Truncated, len(res.Stdout), err)
	}
}
