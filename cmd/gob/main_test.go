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
	err := run([]string{"nope"})
	if !errors.Is(err, errUsage) {
		t.Fatalf("run() error = %v, want errUsage", err)
	}
}

func TestRunClientRoundTrip(t *testing.T) {
	ts := httptest.NewServer(serverpkg.Handler())
	defer ts.Close()

	addr := strings.TrimPrefix(ts.URL, "http://")
	output := captureStdout(t, func() {
		err := runClient([]string{"-addr", addr, "-id", "42", "-type", "ping", "-body", "hello", "-timeout", (50 * time.Millisecond).String()})
		if err != nil {
			t.Fatalf("runClient() unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, `sent    id=42 type=ping body="hello"`) {
		t.Fatalf("runClient() output = %q", output)
	}
	if !strings.Contains(output, `replied id=42 type=ping body="hello"`) {
		t.Fatalf("runClient() output = %q", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}

	os.Stdout = w
	defer func() {
		os.Stdout = original
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.ReadAll() error: %v", err)
	}

	return string(b)
}
