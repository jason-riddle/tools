package main

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunReadsFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "*.xml")
	if err != nil {
		t.Fatalf("os.CreateTemp() error: %v", err)
	}
	if _, err := file.WriteString(`<root><child attr="val">text</child></root>`); err != nil {
		t.Fatalf("WriteString() error: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	stdout, stderr, err := captureOutput(t, func() error {
		return run([]string{file.Name()})
	})
	if err != nil {
		t.Fatalf("run() unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("run() stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "<root>") {
		t.Fatalf("run() stdout = %q, want <root>", stdout)
	}
	if !strings.Contains(stdout, "<child") {
		t.Fatalf("run() stdout = %q, want <child>", stdout)
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
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("run() stdout = %q, want usage text", stdout)
	}
}

func TestRunTooManyArgs(t *testing.T) {
	_, stderr, err := captureOutput(t, func() error {
		return run([]string{"a.xml", "b.xml"})
	})
	if !errors.Is(err, errUsage) {
		t.Fatalf("run() error = %v, want errUsage", err)
	}
	if !strings.Contains(stderr, "accepts at most one file argument") {
		t.Fatalf("run() stderr = %q, want usage error", stderr)
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
