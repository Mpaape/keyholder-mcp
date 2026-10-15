package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"
	"time"

	"keyholder-mcp/internal/ref"
)

// TestMain lets the test binary play the child process of the exit-code
// tests, so they need no shell and no system command.
func TestMain(m *testing.M) {
	switch os.Getenv("KH_EXEC_HELPER") {
	case "exit3":
		os.Exit(3)
	case "sleep":
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestExitCodeIsTheChilds(t *testing.T) {
	if got := exitCode(nil); got != 0 {
		t.Fatalf("nil: got %d, want 0", got)
	}
	if got := exitCode(errors.New("wait failed")); got != 1 {
		t.Fatalf("plain error: got %d, want 1", got)
	}
	c := exec.Command(os.Args[0])
	c.Env = append(os.Environ(), "KH_EXEC_HELPER=exit3")
	if got := exitCode(c.Run()); got != 3 {
		t.Fatalf("child exit 3: got %d, want 3", got)
	}
}

func fakeResolve(r ref.Ref) (string, error) {
	if r.Path == "p/x" && r.Key == "K" {
		return "secret-value", nil
	}
	return "", errors.New("not found")
}

func TestBuildExecEnvInjectsAndFilters(t *testing.T) {
	base := []string{"HOME=/h", "BAO_TOKEN=t", "VAULT_ADDR=a", "KEYHOLDER_SCOPE=p", "API_KEY=old"}
	env, used, err := buildExecEnv(base, []string{"API_KEY=Bearer {{vault://p/x/K}}"}, fakeResolve)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(env)
	want := []string{"API_KEY=Bearer secret-value", "HOME=/h"}
	if strings.Join(env, "|") != strings.Join(want, "|") {
		t.Fatalf("env = %v, want %v", env, want)
	}
	if len(used) != 1 || used[0].String() != "vault://p/x/K" {
		t.Fatalf("used = %v", used)
	}
}

func TestGuardedResolver(t *testing.T) {
	var seen []string
	inner := func(_ context.Context, r ref.Ref) (string, error) {
		seen = append(seen, r.String())
		return "v", nil
	}
	resolve := guardedResolver(context.Background(), "p", map[string]bool{"aws/creds/deploy": true}, inner)
	cases := []struct {
		ref string
		ok  bool
	}{
		{"vault://p/x/K", true},
		{"vault://other/x/K", false},
		{"bao://aws/creds/deploy#access_key", true},
		{"bao://aws/creds/admin#access_key", false},
	}
	for _, c := range cases {
		r, err := ref.Parse(c.ref)
		if err != nil {
			t.Fatal(err)
		}
		_, err = resolve(r)
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok=%v", c.ref, err, c.ok)
		}
	}
	// A refused reference never reaches the vault.
	if strings.Join(seen, ",") != "vault://p/x/K,bao://aws/creds/deploy#access_key" {
		t.Fatalf("inner saw %v", seen)
	}
}

func TestBuildExecEnvRejects(t *testing.T) {
	cases := map[string][]string{
		"no equals":      {"API_KEY"},
		"reserved":       {"BAO_TOKEN={{vault://p/x/K}}"},
		"no placeholder": {"API_KEY=plain"},
		"unresolvable":   {"API_KEY={{vault://p/x/MISSING}}"},
		"bad name":       {"A B={{vault://p/x/K}}"},
	}
	for name, specs := range cases {
		if _, _, err := buildExecEnv(nil, specs, fakeResolve); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

// A malformed spec may carry a literal secret; the error must not echo it.
func TestBuildExecEnvErrorNeverEchoesSpec(t *testing.T) {
	_, _, err := buildExecEnv(nil, []string{"A B=literal-secret"}, fakeResolve)
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), "literal-secret") || strings.Contains(err.Error(), "A B") {
		t.Fatalf("error echoed the spec: %v", err)
	}
}
