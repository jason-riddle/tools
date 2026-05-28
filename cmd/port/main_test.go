package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

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
}

func TestParseStatusOptionsUsesEnv(t *testing.T) {
	t.Setenv("PORTAINER_URL", "http://nas:9000")
	t.Setenv("PORTAINER_API_KEY", "secret")

	fs := flagSetForTest()
	opts, err := parseStatusOptions(fs, nil)
	if err != nil {
		t.Fatalf("parseStatusOptions() unexpected error: %v", err)
	}
	if opts.url != "http://nas:9000" {
		t.Fatalf("parseStatusOptions() url = %q", opts.url)
	}
	if opts.apiKey != "secret" {
		t.Fatalf("parseStatusOptions() apiKey = %q", opts.apiKey)
	}
	if opts.timeout != defaultTimeout {
		t.Fatalf("parseStatusOptions() timeout = %v, want %v", opts.timeout, defaultTimeout)
	}
}

func TestParseStatusOptionsFlagsOverrideEnv(t *testing.T) {
	t.Setenv("PORTAINER_URL", "http://env:9000")
	t.Setenv("PORTAINER_API_KEY", "env-key")

	fs := flagSetForTest()
	opts, err := parseStatusOptions(fs, []string{"--url", "http://flag:9000", "--api-key", "flag-key", "--timeout", "3s", "--json"})
	if err != nil {
		t.Fatalf("parseStatusOptions() unexpected error: %v", err)
	}
	if opts.url != "http://flag:9000" {
		t.Fatalf("parseStatusOptions() url = %q", opts.url)
	}
	if opts.apiKey != "flag-key" {
		t.Fatalf("parseStatusOptions() apiKey = %q", opts.apiKey)
	}
	if opts.timeout != 3*time.Second {
		t.Fatalf("parseStatusOptions() timeout = %v", opts.timeout)
	}
	if !opts.json {
		t.Fatal("parseStatusOptions() json = false, want true")
	}
}

func TestRunStatusJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-Key"); got != "secret-key" {
			t.Fatalf("X-API-Key = %q, want %q", got, "secret-key")
		}
		switch r.URL.Path {
		case "/api/status":
			fmt.Fprint(w, `{"Version":"2.33.4","InstanceID":"instance-1"}`)
		case "/api/endpoints":
			fmt.Fprint(w, `[{"Id":2,"Name":"local","Type":1,"URL":"unix:///var/run/docker.sock","PublicURL":"http://nas","Status":1,"Snapshots":[{"Time":1710000000,"DockerVersion":"24.0.2","TotalCPU":2,"TotalMemory":2048,"ContainerCount":12,"RunningContainerCount":10,"StoppedContainerCount":2,"HealthyContainerCount":0,"UnhealthyContainerCount":0,"VolumeCount":111,"ImageCount":15,"ServiceCount":0,"StackCount":8}]}]`)
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	stdout, stderr, err := captureOutput(t, func() error {
		return runStatus([]string{"--url", server.URL, "--api-key", "secret-key", "--json"})
	})
	if err != nil {
		t.Fatalf("runStatus() unexpected error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("runStatus() stderr = %q, want empty", stderr)
	}

	var out statusOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error: %v\nstdout=%q", err, stdout)
	}
	if out.Version != "2.33.4" {
		t.Fatalf("runStatus() version = %q", out.Version)
	}
	if out.EndpointCount != 1 {
		t.Fatalf("runStatus() endpoint_count = %d", out.EndpointCount)
	}
	if len(out.Endpoints) != 1 || out.Endpoints[0].Name != "local" {
		t.Fatalf("runStatus() endpoints = %+v", out.Endpoints)
	}
}

func TestRunStatusMissingConfigReturnsUsage(t *testing.T) {
	t.Setenv("PORTAINER_URL", "")
	t.Setenv("PORTAINER_API_KEY", "")

	_, stderr, err := captureOutput(t, func() error {
		return runStatus(nil)
	})
	if !errors.Is(err, errUsage) {
		t.Fatalf("runStatus() error = %v, want errUsage", err)
	}
	if !strings.Contains(stderr, "missing Portainer URL") {
		t.Fatalf("runStatus() stderr = %q", stderr)
	}
	if !strings.Contains(stderr, "PORTAINER_URL") {
		t.Fatalf("runStatus() stderr = %q", stderr)
	}
}

func flagSetForTest() *flag.FlagSet {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
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
