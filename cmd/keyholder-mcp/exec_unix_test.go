//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// A child killed by a signal yields 128 + signal number, the shell
// convention, so a supervisor sees the same thing it would from a shell.
// The child is the test binary itself (see TestMain).
func TestExitCodeOnSignal(t *testing.T) {
	c := exec.Command(os.Args[0])
	c.Env = append(os.Environ(), "KH_EXEC_HELPER=sleep")
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if err := c.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	want := 128 + int(syscall.SIGTERM)
	if got := exitCode(c.Wait()); got != want {
		t.Fatalf("SIGTERM: got %d, want %d", got, want)
	}
}
