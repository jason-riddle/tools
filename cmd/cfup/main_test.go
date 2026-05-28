package main

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCLIUnhealthyAccessPageExitsNonZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		w.Header().Set("Cf-Access-Domain", host)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "Sign in · Cloudflare Access")
	}))
	defer server.Close()

	stdout, stderr, code := runCLI(t, nil, server.URL)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout, "unhealthy status=200") {
		t.Fatalf("stdout = %q, want unhealthy status", stdout)
	}
	if !strings.Contains(stdout, "final_url="+server.URL) {
		t.Fatalf("stdout = %q, want final URL", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestCLIHealthyWithServiceTokenExitsZero(t *testing.T) {
	const (
		clientID     = "client-id"
		clientSecret = "client-secret"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			if got := r.Header.Get("CF-Access-Client-Id"); got != clientID {
				t.Fatalf("CF-Access-Client-Id = %q, want %q", got, clientID)
			}
			if got := r.Header.Get("CF-Access-Client-Secret"); got != clientSecret {
				t.Fatalf("CF-Access-Client-Secret = %q, want %q", got, clientSecret)
			}
			http.Redirect(w, r, "/accounts/login/?next=/", http.StatusFound)
		case "/accounts/login/":
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "Paperless-ngx sign in")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	stdout, stderr, code := runCLI(t, map[string]string{
		"CF_ACCESS_CLIENT_ID":     clientID,
		"CF_ACCESS_CLIENT_SECRET": clientSecret,
	}, server.URL)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q stdout=%q", code, stderr, stdout)
	}
	if !strings.Contains(stdout, "healthy status=200") {
		t.Fatalf("stdout = %q, want healthy status", stdout)
	}
	if !strings.Contains(stdout, "/accounts/login/?next=/") {
		t.Fatalf("stdout = %q, want final redirect target", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestCLIMissingURLExitsNonZero(t *testing.T) {
	stdout, stderr, code := runCLI(t, nil)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "missing required --url or positional URL") {
		t.Fatalf("stderr = %q, want missing URL error", stderr)
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr = %q, want usage text", stderr)
	}
}

func TestCLIPartialTokenEnvExitsNonZero(t *testing.T) {
	stdout, stderr, code := runCLI(t, map[string]string{
		"CF_ACCESS_CLIENT_ID":     "client-id",
		"CF_ACCESS_CLIENT_SECRET": "",
	}, "https://example.com")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "CF_ACCESS_CLIENT_ID and CF_ACCESS_CLIENT_SECRET must both be set") {
		t.Fatalf("stderr = %q, want partial env error", stderr)
	}
}

func runCLI(t *testing.T, env map[string]string, args ...string) (string, string, int) {
	t.Helper()

	cmdArgs := append([]string{"-test.run=TestHelperProcess", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() error: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("StderrPipe() error: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	stdoutBytes, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("ReadAll(stdout) error: %v", err)
	}
	stderrBytes, err := io.ReadAll(stderr)
	if err != nil {
		t.Fatalf("ReadAll(stderr) error: %v", err)
	}

	exitCode := 0
	if err := cmd.Wait(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("Wait() error = %v", err)
		}
		exitCode = exitErr.ExitCode()
	}

	return string(stdoutBytes), string(stderrBytes), exitCode
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}

	log.SetFlags(0)
	log.SetPrefix("cfup: ")

	err := run(args)
	if err == nil {
		os.Exit(0)
	}

	var quiet quietError
	if !errors.Is(err, errUsage) && !errors.As(err, &quiet) {
		log.Print(err)
	}
	os.Exit(1)
}
