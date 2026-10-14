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

// The test binary itself plays the command being run, which works the same on
// Linux, macOS and Windows. See TestMain.
const helperEnv = "KH_TEST_HELPER"

// helperDir holds a COPY of the test binary named kh-helper. Not a hard link:
// on Windows a running executable cannot be deleted, and a link would point at
// the running test binary, so cleaning up the directory would fail.
var helperDir string

func TestMain(m *testing.M) {
	if mode := os.Getenv(helperEnv); mode != "" {
		switch mode {
		case "echo": // prints the arguments and the API_KEY variable
			fmt.Println(strings.Join(os.Args[1:], " "))
			fmt.Fprintln(os.Stderr, "API_KEY="+os.Getenv("API_KEY"))
		case "env": // prints the names of inherited variables
			for _, kv := range os.Environ() {
				name, _, _ := strings.Cut(kv, "=")
				fmt.Println(name)
			}
		case "cloud": // prints the cloud variables and the token file it was given
			for _, k := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN", "AWS_PROFILE", "GOOGLE_OAUTH_ACCESS_TOKEN"} {
				v, ok := os.LookupEnv(k)
				fmt.Printf("%s=%v:%s\n", k, ok, v)
			}
			if f := os.Getenv("CLOUDSDK_AUTH_ACCESS_TOKEN_FILE"); f != "" {
				b, err := os.ReadFile(f)
				fmt.Printf("TOKEN_FILE=%s err=%v\n", b, err)
				fmt.Fprintln(os.Stderr, f)
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
		fmt.Fprintln(os.Stderr, "prepare kh-helper:", err)
		os.Exit(1)
	}
	helperDir = dir
	code := m.Run()
	_ = os.RemoveAll(dir) // best effort: antivirus may hold the file for a moment
	os.Exit(code)
}

// prepareHelper copies the test binary into a temporary directory.
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
		"vault://proj/dev/TOKEN": "tok-SUPER-secret",
		"vault://proj/dev/KEY":   "key-123456",
		"vault://other/dev/X":    "out-of-scope",
	}
	return &Runner{
		Allow: map[string]bool{"kh-helper": true},
		Scope: "proj",
		Resolve: func(_ context.Context, r ref.Ref) (string, error) {
			v, ok := secrets[r.String()]
			if !ok {
				return "", errors.New("not found")
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
	if strings.Contains(res.Stdout+res.Stderr, "SUPER-secret") || strings.Contains(res.Stdout+res.Stderr, "key-123456") {
		t.Fatalf("value leaked: stdout=%q stderr=%q", res.Stdout, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "Bearer [hidden:vault://proj/dev/TOKEN] literal") {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	if !strings.Contains(res.Stderr, "API_KEY=[hidden:vault://proj/dev/KEY]") {
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
	_, err := r.Run(context.Background(), Request{Command: "kh-helper", Args: []string{"{{vault://other/dev/X}}"}})
	if !errors.Is(err, ErrOutOfScope) {
		t.Fatalf("err = %v, want ErrOutOfScope", err)
	}
}

func TestRunStripsVaultEnv(t *testing.T) {
	r := helper(t, "env", "BAO_TOKEN=s.must-not-leak", "VAULT_TOKEN=x", "KEYHOLDER_SCOPE=proj", "HOME_OK=1")
	res, err := r.Run(context.Background(), Request{Command: "kh-helper"})
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"BAO_TOKEN", "VAULT_TOKEN", "KEYHOLDER_SCOPE"} {
		if strings.Contains(res.Stdout, bad) {
			t.Errorf("%s reached the child process", bad)
		}
	}
	if !strings.Contains(res.Stdout, "HOME_OK") {
		t.Errorf("a regular variable was removed: %q", res.Stdout)
	}
	if _, err := r.Run(context.Background(), Request{Command: "kh-helper", Env: map[string]string{"BAO_TOKEN": "x"}}); err == nil {
		t.Error("expected BAO_TOKEN in env to be refused")
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

// fakeSession serves vault:// and bao:// values and records what a run did.
type fakeSession struct {
	values   map[string]string // by reference string
	reads    map[string]int    // dynamic reads by path
	finished int
}

func (f *fakeSession) Resolve(_ context.Context, r ref.Ref) (string, error) {
	if r.Dynamic {
		f.reads[r.Path]++
	}
	v, ok := f.values[r.String()]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func (f *fakeSession) Finish(context.Context) []string {
	f.finished++
	return []string{"revoked 1 dynamic credential lease(s) after the run"}
}

func cloudRunner(t *testing.T, sess *fakeSession, extraEnv ...string) *Runner {
	r := helper(t, "cloud", extraEnv...)
	r.Resolve = nil
	r.Dynamic = map[string]bool{"aws/creds/deploy": true, "gcp/roleset/viewer/token": true}
	r.Begin = func() Session { return sess }
	return r
}

func TestRunAWSCredentials(t *testing.T) {
	sess := &fakeSession{reads: map[string]int{}, values: map[string]string{
		"bao://aws/creds/deploy#access_key": "AKIAFAKE0000000001",
		"bao://aws/creds/deploy#secret_key": "fake-secret-key-0001",
	}}
	// An inherited profile and session token must not mix with the new key.
	r := cloudRunner(t, sess, "AWS_PROFILE=prod", "AWS_SESSION_TOKEN=stale-token")
	res, err := r.Run(context.Background(), Request{Command: "kh-helper", Credentials: []string{"aws:aws/creds/deploy"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"AWS_ACCESS_KEY_ID=true:[hidden:bao://aws/creds/deploy#access_key]",
		"AWS_SECRET_ACCESS_KEY=true:[hidden:bao://aws/creds/deploy#secret_key]",
		"AWS_SESSION_TOKEN=false:",
		"AWS_PROFILE=false:",
	} {
		if !strings.Contains(res.Stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, res.Stdout)
		}
	}
	if strings.Contains(res.Stdout, "AKIAFAKE") || strings.Contains(res.Stdout, "fake-secret") {
		t.Errorf("value leaked: %s", res.Stdout)
	}
	if sess.reads["aws/creds/deploy"] < 1 || sess.finished != 1 || len(res.Notes) != 1 {
		t.Errorf("reads=%v finished=%d notes=%v", sess.reads, sess.finished, res.Notes)
	}
}

func TestRunGCPTokenFile(t *testing.T) {
	sess := &fakeSession{reads: map[string]int{}, values: map[string]string{
		"bao://gcp/roleset/viewer/token#token": "ya29.fake-access-token",
	}}
	r := cloudRunner(t, sess)
	res, err := r.Run(context.Background(), Request{Command: "kh-helper", Credentials: []string{"gcp:gcp/roleset/viewer/token"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Stdout, "TOKEN_FILE=[hidden:bao://gcp/roleset/viewer/token#token] err=<nil>") {
		t.Errorf("token file not passed:\n%s", res.Stdout)
	}
	path := strings.TrimSpace(res.Stderr)
	if path == "" {
		t.Fatal("no token file path")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("token file %s still exists after the run (err=%v)", path, err)
	}
}

func TestRunDynamicNotAllowed(t *testing.T) {
	sess := &fakeSession{reads: map[string]int{}, values: map[string]string{}}
	r := cloudRunner(t, sess)
	_, err := r.Run(context.Background(), Request{Command: "kh-helper", Args: []string{"{{bao://aws/creds/admin#access_key}}"}})
	if !errors.Is(err, ErrDynamicNotAllow) {
		t.Fatalf("err = %v, want ErrDynamicNotAllow", err)
	}
	if sess.reads["aws/creds/admin"] != 0 {
		t.Error("a path outside KEYHOLDER_DYNAMIC was read")
	}
	if sess.finished != 1 {
		t.Errorf("finished = %d, want 1: a refused run still releases what it created", sess.finished)
	}
}

func TestExpandCredentialsConflicts(t *testing.T) {
	for _, req := range []Request{
		{Credentials: []string{"azure:x/y"}},
		{Credentials: []string{"aws:"}},
		{Credentials: []string{"aws:a/../b"}},
		{Credentials: []string{"aws:aws/creds/a"}, Env: map[string]string{"AWS_ACCESS_KEY_ID": "x"}},
		{Env: map[string]string{"A": "1"}, Files: map[string]string{"A": "2"}},
	} {
		if _, _, _, _, err := expandCredentials(req); err == nil {
			t.Errorf("expandCredentials(%+v) = nil error", req)
		}
	}
}
