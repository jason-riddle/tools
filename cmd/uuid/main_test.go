package main

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	uuidpkg "github.com/jason-riddle/tools/internal/uuid"
)

func TestRunNewPrintsValidUUID(t *testing.T) {
	stdout, _, err := captureOutput(t, func() error {
		return runNew([]string{"-v", "4"})
	})
	if err != nil {
		t.Fatalf("runNew() unexpected error: %v", err)
	}

	value := strings.TrimSpace(stdout)
	parsed, err := uuidpkg.Parse(value)
	if err != nil {
		t.Fatalf("Parse(%q) unexpected error: %v", value, err)
	}
	if parsed.Version() != 4 {
		t.Fatalf("runNew() printed version %d UUID, want 4", parsed.Version())
	}
}

func TestRunVersionPrintsUUIDVersion(t *testing.T) {
	stdout, _, err := captureOutput(t, func() error {
		return runVersion([]string{"f81d4fae-7dec-11d0-a765-00a0c91e6bf6"})
	})
	if err != nil {
		t.Fatalf("runVersion() unexpected error: %v", err)
	}

	if got := strings.TrimSpace(stdout); got != "1" {
		t.Fatalf("runVersion() output = %q, want %q", got, "1")
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
	if !strings.Contains(stdout, "Run 'uuid <command> -h'") {
		t.Fatalf("run() stdout = %q", stdout)
	}
}

func TestRunParseUsageErrorIncludesMessageAndUsage(t *testing.T) {
	_, stderr, err := captureOutput(t, func() error {
		return runParse(nil)
	})
	if !errors.Is(err, errUsage) {
		t.Fatalf("runParse() error = %v, want errUsage", err)
	}
	if !strings.Contains(stderr, "uuid parse: expected exactly one uuid-string argument") {
		t.Fatalf("runParse() stderr = %q", stderr)
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Fatalf("runParse() stderr = %q", stderr)
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
