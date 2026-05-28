package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseOptionsOffset(t *testing.T) {
	opts, err := parseOptions([]string{"--nano", "+24h"})
	if err != nil {
		t.Fatalf("parseOptions() unexpected error: %v", err)
	}
	if !opts.nano {
		t.Fatal("parseOptions() did not set nano mode")
	}
	if !opts.hasOffset || opts.offset != 24*time.Hour {
		t.Fatalf("parseOptions() offset = %v, hasOffset = %v", opts.offset, opts.hasOffset)
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

func TestLocationFromEnvDefaultUTC(t *testing.T) {
	loc, err := locationFromEnv("", false)
	if err != nil {
		t.Fatalf("locationFromEnv() unexpected error: %v", err)
	}
	if loc != time.UTC {
		t.Fatalf("locationFromEnv() = %v, want UTC", loc)
	}
}

func TestLocationFromEnvLoadsLocation(t *testing.T) {
	loc, err := locationFromEnv("America/New_York", true)
	if err != nil {
		t.Fatalf("locationFromEnv() unexpected error: %v", err)
	}
	if got := loc.String(); got != "America/New_York" {
		t.Fatalf("locationFromEnv() = %q, want %q", got, "America/New_York")
	}
}

func TestFormatTimeDefaultRFC3339(t *testing.T) {
	ts := time.Date(2026, time.May, 27, 15, 4, 5, 123456789, time.UTC)
	got, err := formatTime(ts, options{})
	if err != nil {
		t.Fatalf("formatTime() unexpected error: %v", err)
	}
	if want := "2026-05-27T15:04:05Z"; got != want {
		t.Fatalf("formatTime() = %q, want %q", got, want)
	}
}

func TestFormatTimeJSON(t *testing.T) {
	ts := time.Date(2026, time.May, 27, 15, 4, 5, 123456789, time.UTC)
	got, err := formatTime(ts, options{json: true})
	if err != nil {
		t.Fatalf("formatTime() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(got), &data); err != nil {
		t.Fatalf("json.Unmarshal() unexpected error: %v", err)
	}

	if data["RFC3339"] != "2026-05-27T15:04:05Z" {
		t.Fatalf("RFC3339 = %v, want %q", data["RFC3339"], "2026-05-27T15:04:05Z")
	}
	if data["epoch"] != float64(ts.Unix()) {
		t.Fatalf("epoch = %v, want %v", data["epoch"], ts.Unix())
	}
	if _, ok := data["DateTime"]; !ok {
		t.Fatal("DateTime missing from JSON output")
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
