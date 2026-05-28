package main

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunHelpPrintsUsageToStdout(t *testing.T) {
	stdout, stderr, err := captureOutput(t, func() error {
		return run([]string{"-h"}, strings.NewReader(""))
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
}

func TestRunStatusMissingEnvIncludesAllVars(t *testing.T) {
	t.Setenv("NAS_HOST", "")
	t.Setenv("NAS_USER", "")
	t.Setenv("NAS_PASSWORD", "")

	_, stderr, err := captureOutput(t, func() error {
		return runStatus(nil)
	})
	if !errors.Is(err, errUsage) {
		t.Fatalf("runStatus() error = %v, want errUsage", err)
	}
	for _, name := range []string{"NAS_HOST", "NAS_USER", "NAS_PASSWORD"} {
		if !strings.Contains(stderr, name) {
			t.Fatalf("runStatus() stderr = %q, want %q", stderr, name)
		}
	}
}

func TestRunRebootCancelled(t *testing.T) {
	t.Setenv("NAS_HOST", "nas")
	t.Setenv("NAS_USER", "jason")
	t.Setenv("NAS_PASSWORD", "secret")

	stdout, stderr, err := captureOutput(t, func() error {
		return runReboot(nil, strings.NewReader("n\n"))
	})
	if err == nil || err.Error() != "reboot cancelled" {
		t.Fatalf("runReboot() error = %v, want reboot cancelled", err)
	}
	if stderr != "" {
		t.Fatalf("runReboot() stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Reboot nas? [y/N]") {
		t.Fatalf("runReboot() stdout = %q", stdout)
	}
}

func TestRunRebootHelpPrintsUsage(t *testing.T) {
	stdout, stderr, err := captureOutput(t, func() error {
		return runReboot([]string{"-h"}, strings.NewReader(""))
	})
	if err != nil {
		t.Fatalf("runReboot() unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("runReboot() stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "-confirm") {
		t.Fatalf("runReboot() stdout = %q", stdout)
	}
}

func TestFormatUptime(t *testing.T) {
	if got := formatUptime(14*24*3600 + 3*3600 + 22*60); got != "14d 3h 22m" {
		t.Fatalf("formatUptime() = %q, want %q", got, "14d 3h 22m")
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
