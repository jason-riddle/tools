package main

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	serverpkg "github.com/jason-riddle/tools/internal/gob/server"
	"net/http/httptest"
)

func TestRunUnknownCommand(t *testing.T) {
	_, stderr, err := captureOutput(t, func() error {
		return run([]string{"nope"})
	})
	if !errors.Is(err, errUsage) {
		t.Fatalf("run() error = %v, want errUsage", err)
	}
	if !strings.Contains(stderr, `gob: unknown command "nope"`) {
		t.Fatalf("run() stderr = %q", stderr)
	}
}

func TestRunClientRoundTrip(t *testing.T) {
	ts := httptest.NewServer(serverpkg.Handler())
	defer ts.Close()

	addr := strings.TrimPrefix(ts.URL, "http://")
	stdout, _, err := captureOutput(t, func() error {
		return runClient([]string{"-addr", addr, "-id", "42", "-type", "ping", "-body", "hello", "-timeout", (50 * time.Millisecond).String()})
	})
	if err != nil {
		t.Fatalf("runClient() unexpected error: %v", err)
	}

	if !strings.Contains(stdout, `sent    id=42 type=ping body="hello"`) {
		t.Fatalf("runClient() output = %q", stdout)
	}
	if !strings.Contains(stdout, `replied id=42 type=ping body="hello"`) {
		t.Fatalf("runClient() output = %q", stdout)
	}
}

func TestRunHelpPrintsUsageToStdout(t *testing.T) {
	stdout, stderr, err := captureOutput(t, func() error {
		return run([]string{"-h"})
	})
	if err != nil {
		t.Fatalf("run() unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("run() stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Commands:") {
		t.Fatalf("run() stdout = %q", stdout)
	}
	if !strings.Contains(stdout, "Run 'gob <command> -h'") {
		t.Fatalf("run() stdout = %q", stdout)
	}
}

func TestRunClientHelpPrintsSubcommandUsage(t *testing.T) {
	stdout, stderr, err := captureOutput(t, func() error {
		return runClient([]string{"-h"})
	})
	if err != nil {
		t.Fatalf("runClient() unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("runClient() stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "gob client - send a gob message to the server") {
		t.Fatalf("runClient() stdout = %q", stdout)
	}
}

func captureOutput(t *testing.T, fn func() error) (string, string, error) {
	t.Helper()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() stdout error: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() stderr error: %v", err)
	}

	originalStdout := os.Stdout
	originalStderr := os.Stderr
	os.Stdout = stdoutW
	os.Stderr = stderrW
	defer func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	}()

	runErr := fn()

	if err := stdoutW.Close(); err != nil {
		t.Fatalf("stdout Close() error: %v", err)
	}
	if err := stderrW.Close(); err != nil {
		t.Fatalf("stderr Close() error: %v", err)
	}

	stdoutBytes, err := io.ReadAll(stdoutR)
	if err != nil {
		t.Fatalf("io.ReadAll(stdout) error: %v", err)
	}
	stderrBytes, err := io.ReadAll(stderrR)
	if err != nil {
		t.Fatalf("io.ReadAll(stderr) error: %v", err)
	}

	return string(stdoutBytes), string(stderrBytes), runErr
}
