package main

import (
	"io"
	"os"
	"strings"
	"testing"

	uuidpkg "github.com/jason-riddle/tools/internal/uuid"
)

func TestRunNewPrintsValidUUID(t *testing.T) {
	output := captureStdout(t, func() {
		if err := runNew([]string{"-v", "4"}); err != nil {
			t.Fatalf("runNew() unexpected error: %v", err)
		}
	})

	value := strings.TrimSpace(output)
	parsed, err := uuidpkg.Parse(value)
	if err != nil {
		t.Fatalf("Parse(%q) unexpected error: %v", value, err)
	}
	if parsed.Version() != 4 {
		t.Fatalf("runNew() printed version %d UUID, want 4", parsed.Version())
	}
}

func TestRunVersionPrintsUUIDVersion(t *testing.T) {
	output := captureStdout(t, func() {
		if err := runVersion([]string{"f81d4fae-7dec-11d0-a765-00a0c91e6bf6"}); err != nil {
			t.Fatalf("runVersion() unexpected error: %v", err)
		}
	})

	if got := strings.TrimSpace(output); got != "1" {
		t.Fatalf("runVersion() output = %q, want %q", got, "1")
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
