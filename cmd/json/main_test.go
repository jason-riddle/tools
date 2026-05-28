package main

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunSortsObjectKeys(t *testing.T) {
	input := `{"z": 1, "a": 2, "m": 3}`
	stdout, _, err := captureOutput(t, func() error {
		return writeJSON(os.Stdout, strings.NewReader(input), options{})
	})
	if err != nil {
		t.Fatalf("runWithReader() unexpected error: %v", err)
	}
	// Keys must appear in sorted order: a, m, z
	aIdx := strings.Index(stdout, `"a"`)
	mIdx := strings.Index(stdout, `"m"`)
	zIdx := strings.Index(stdout, `"z"`)
	if aIdx < 0 || mIdx < 0 || zIdx < 0 {
		t.Fatalf("runWithReader() output = %q, missing expected keys", stdout)
	}
	if !(aIdx < mIdx && mIdx < zIdx) {
		t.Fatalf("runWithReader() keys not sorted: output = %q", stdout)
	}
}

func TestRunNestedObjectKeysSorted(t *testing.T) {
	input := `{"outer": {"z": 1, "a": 2}}`
	stdout, _, err := captureOutput(t, func() error {
		return writeJSON(os.Stdout, strings.NewReader(input), options{})
	})
	if err != nil {
		t.Fatalf("runWithReader() unexpected error: %v", err)
	}
	aIdx := strings.Index(stdout, `"a"`)
	zIdx := strings.Index(stdout, `"z"`)
	if aIdx < 0 || zIdx < 0 {
		t.Fatalf("runWithReader() output = %q, missing nested keys", stdout)
	}
	if aIdx >= zIdx {
		t.Fatalf("runWithReader() nested keys not sorted: output = %q", stdout)
	}
}

func TestRunCompact(t *testing.T) {
	input := `{"b": 2, "a": 1}`
	stdout, _, err := captureOutput(t, func() error {
		return writeJSON(os.Stdout, strings.NewReader(input), options{compact: true})
	})
	if err != nil {
		t.Fatalf("runWithReader() unexpected error: %v", err)
	}
	line := strings.TrimSpace(stdout)
	if strings.Contains(line, "\n") {
		t.Fatalf("runWithReader() compact output contains newlines: %q", stdout)
	}
	if line != `{"a":1,"b":2}` {
		t.Fatalf("runWithReader() compact output = %q, want %q", line, `{"a":1,"b":2}`)
	}
}

func TestRunPreservesArrayOrder(t *testing.T) {
	input := `{"arr": [3, 1, 2]}`
	stdout, _, err := captureOutput(t, func() error {
		return writeJSON(os.Stdout, strings.NewReader(input), options{compact: true})
	})
	if err != nil {
		t.Fatalf("runWithReader() unexpected error: %v", err)
	}
	line := strings.TrimSpace(stdout)
	if line != `{"arr":[3,1,2]}` {
		t.Fatalf("runWithReader() array order not preserved: %q", line)
	}
}

func TestRunSortArraysFlag(t *testing.T) {
	input := `{"arr": [3, 1, 2]}`
	stdout, _, err := captureOutput(t, func() error {
		return writeJSON(os.Stdout, strings.NewReader(input), options{sortArrays: true, compact: true})
	})
	if err != nil {
		t.Fatalf("runWithReader() unexpected error: %v", err)
	}
	line := strings.TrimSpace(stdout)
	if line != `{"arr":[1,2,3]}` {
		t.Fatalf("runWithReader() sort-arrays output = %q, want %q", line, `{"arr":[1,2,3]}`)
	}
}

func TestRunSortArraysPreservesObjectArrayOrder(t *testing.T) {
	// Arrays containing objects must not be reordered.
	input := `{"arr": [{"b": 2}, {"a": 1}]}`
	stdout, _, err := captureOutput(t, func() error {
		return writeJSON(os.Stdout, strings.NewReader(input), options{sortArrays: true, compact: true})
	})
	if err != nil {
		t.Fatalf("runWithReader() unexpected error: %v", err)
	}
	line := strings.TrimSpace(stdout)
	// b-object comes before a-object (original order preserved)
	bIdx := strings.Index(line, `"b"`)
	aIdx := strings.Index(line, `"a"`)
	if bIdx < 0 || aIdx < 0 {
		t.Fatalf("runWithReader() output = %q, missing expected keys", line)
	}
	if bIdx >= aIdx {
		t.Fatalf("runWithReader() object-array order not preserved: %q", line)
	}
}

func TestRunInvalidJSON(t *testing.T) {
	input := `{invalid}`
	_, stderr, err := captureOutput(t, func() error {
		return writeJSON(os.Stdout, strings.NewReader(input), options{})
	})
	if err == nil {
		t.Fatal("writeJSON() expected error for invalid JSON")
	}
	if errors.Is(err, errUsage) {
		t.Fatal("writeJSON() should not return errUsage for parse error")
	}
	_ = stderr // error is returned, not printed to stderr by helper
}

func TestRunReadsFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "*.json")
	if err != nil {
		t.Fatalf("os.CreateTemp() error: %v", err)
	}
	if _, err := file.WriteString(`{"b":2,"a":1}`); err != nil {
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
	if !strings.Contains(stdout, "\"a\": 1") || !strings.Contains(stdout, "\"b\": 2") {
		t.Fatalf("run() stdout = %q, want sorted JSON", stdout)
	}
}

func TestScalarSortKey(t *testing.T) {
	if got := compareScalars(nil, false); got >= 0 {
		t.Fatalf("compareScalars(nil, false) = %d, want < 0", got)
	}
	if got := compareScalars(false, true); got >= 0 {
		t.Fatalf("compareScalars(false, true) = %d, want < 0", got)
	}
	if got := compareScalars("abc", "abd"); got >= 0 {
		t.Fatalf("compareScalars(abc, abd) = %d, want < 0", got)
	}
}

func TestSortArraysMixedScalarTypes(t *testing.T) {
	stdout, _, err := captureOutput(t, func() error {
		return writeJSON(os.Stdout, strings.NewReader(`{"arr":["b",null,false,2,"a"]}`), options{sortArrays: true, compact: true})
	})
	if err != nil {
		t.Fatalf("writeJSON() unexpected error: %v", err)
	}
	if got := strings.TrimSpace(stdout); got != `{"arr":[null,false,2,"a","b"]}` {
		t.Fatalf("writeJSON() output = %q, want %q", got, `{"arr":[null,false,2,"a","b"]}`)
	}
}

func TestSortArraysNumericOrder(t *testing.T) {
	stdout, _, err := captureOutput(t, func() error {
		return writeJSON(os.Stdout, strings.NewReader(`{"arr":[2,10,1]}`), options{sortArrays: true, compact: true})
	})
	if err != nil {
		t.Fatalf("writeJSON() unexpected error: %v", err)
	}
	if got := strings.TrimSpace(stdout); got != `{"arr":[1,2,10]}` {
		t.Fatalf("writeJSON() output = %q, want %q", got, `{"arr":[1,2,10]}`)
	}
}

func TestWriteJSONPreservesLargeInteger(t *testing.T) {
	stdout, _, err := captureOutput(t, func() error {
		return writeJSON(os.Stdout, strings.NewReader(`{"n":9007199254740993}`), options{compact: true})
	})
	if err != nil {
		t.Fatalf("writeJSON() unexpected error: %v", err)
	}
	if got := strings.TrimSpace(stdout); got != `{"n":9007199254740993}` {
		t.Fatalf("writeJSON() output = %q, want %q", got, `{"n":9007199254740993}`)
	}
}

func TestWriteJSONRejectsTrailingContent(t *testing.T) {
	_, _, err := captureOutput(t, func() error {
		return writeJSON(os.Stdout, strings.NewReader(`{"a":1} trailing`), options{})
	})
	if err == nil {
		t.Fatal("writeJSON() expected error for trailing content")
	}
	if !strings.Contains(err.Error(), "extra content") && !strings.Contains(err.Error(), "invalid character") {
		t.Fatalf("writeJSON() error = %q, want trailing-content parse error", err)
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
		return run([]string{"a.json", "b.json"})
	})
	if !errors.Is(err, errUsage) {
		t.Fatalf("run() error = %v, want errUsage", err)
	}
	if !strings.Contains(stderr, "accepts at most one file argument") {
		t.Fatalf("run() stderr = %q, want usage error", stderr)
	}
}

// runWithReader is a test helper that runs the core logic with a given reader
// and options, bypassing file and stdin handling.
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
