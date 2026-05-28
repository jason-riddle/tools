package main

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	tickformat "github.com/jason-riddle/tools/internal/tick/format"
)

func TestParseOptionsOffset(t *testing.T) {
	opts, err := parseOptions([]string{"--nano", "+24h"})
	if err != nil {
		t.Fatalf("parseOptions() unexpected error: %v", err)
	}
	if opts.mode != tickformat.ModeRFC3339Nano {
		t.Fatalf("parseOptions() mode = %v, want nano mode", opts.mode)
	}
	if opts.offset != 24*time.Hour {
		t.Fatalf("parseOptions() offset = %v, want %v", opts.offset, 24*time.Hour)
	}
}

func TestParseOptionsRejectsMultipleModes(t *testing.T) {
	_, err := parseOptions([]string{"--nano", "--epoch"})
	if err == nil {
		t.Fatal("parseOptions() expected an error for multiple output modes")
	}
	if !strings.Contains(err.Error(), "-nano, -epoch, -format, and -json are mutually exclusive") {
		t.Fatalf("parseOptions() error = %q", err)
	}
}

func TestParseOptionsRejectsMultipleOffsets(t *testing.T) {
	_, err := parseOptions([]string{"+1h", "+30m"})
	if err == nil {
		t.Fatal("parseOptions() expected an error for multiple offsets")
	}
}

func TestRunHelpPrintsUsageToStdout(t *testing.T) {
	stdout, stderr := captureOutput(t, func() error {
		return run([]string{"-h"})
	})

	if stderr != "" {
		t.Fatalf("run() stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("run() stdout = %q, want usage text", stdout)
	}
	if !strings.Contains(stdout, "Notes:") {
		t.Fatalf("run() stdout = %q, want notes section", stdout)
	}
}

func TestRunUsageErrorIncludesMessageAndUsage(t *testing.T) {
	_, stderr := captureOutput(t, func() error {
		return run([]string{"--bogus"})
	})

	if !strings.Contains(stderr, "tick: flag provided but not defined: -bogus") {
		t.Fatalf("run() stderr = %q, want parse error prefix", stderr)
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Fatalf("run() stderr = %q, want usage text", stderr)
	}
}

func captureOutput(t *testing.T, fn func() error) (string, string) {
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

	if runErr != nil && !errors.Is(runErr, errUsage) {
		t.Fatalf("run() unexpected error: %v", runErr)
	}

	return string(stdoutBytes), string(stderrBytes)
}
