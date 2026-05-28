package main

import (
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"net/http"
)

func TestParseOptionsDefaults(t *testing.T) {
	opts, err := parseOptions(nil)
	if err != nil {
		t.Fatalf("parseOptions() unexpected error: %v", err)
	}

	if opts.user != defaultUser {
		t.Fatalf("parseOptions() user = %q, want %q", opts.user, defaultUser)
	}
	if opts.timeout != defaultTimeout {
		t.Fatalf("parseOptions() timeout = %v, want %v", opts.timeout, defaultTimeout)
	}
}

func TestParseOptionsAllowsTimeoutBeforeUser(t *testing.T) {
	opts, err := parseOptions([]string{"--timeout", "3s", "octocat"})
	if err != nil {
		t.Fatalf("parseOptions() unexpected error: %v", err)
	}

	if opts.user != "octocat" {
		t.Fatalf("parseOptions() user = %q, want %q", opts.user, "octocat")
	}
	if opts.timeout != 3*time.Second {
		t.Fatalf("parseOptions() timeout = %v, want %v", opts.timeout, 3*time.Second)
	}
}

func TestParseOptionsRejectsMultipleUsers(t *testing.T) {
	_, err := parseOptions([]string{"octocat", "hubot"})
	if err == nil {
		t.Fatal("parseOptions() expected an error for multiple usernames")
	}
}

func TestRunPrintsFetchedKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ssh-rsa AAAATEST default\n")
	}))
	defer server.Close()

	originalBaseURL := githubBaseURL
	githubBaseURL = server.URL
	defer func() {
		githubBaseURL = originalBaseURL
	}()

	stdout, stderr := captureOutput(t, func() error {
		return run(nil)
	})

	if stderr != "" {
		t.Fatalf("run() stderr = %q, want empty", stderr)
	}
	if stdout != "ssh-rsa AAAATEST default\n" {
		t.Fatalf("run() stdout = %q", stdout)
	}
}

func TestRunReturnsUsageOnTooManyArgs(t *testing.T) {
	_, stderr := captureOutput(t, func() error {
		return run([]string{"one", "two"})
	})

	if !strings.Contains(stderr, "accepts at most one username argument") {
		t.Fatalf("run() stderr = %q, want parse error prefix", stderr)
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Fatalf("run() stderr = %q, want usage text", stderr)
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
	if !strings.Contains(stdout, "Arguments:") {
		t.Fatalf("run() stdout = %q, want arguments section", stdout)
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
