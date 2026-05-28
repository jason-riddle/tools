package format

import (
	"strings"
	"testing"
)

func TestWriteSortsObjectKeys(t *testing.T) {
	input := `{"z": 1, "a": 2, "m": 3}`
	var out strings.Builder
	if err := Write(&out, strings.NewReader(input), Options{}); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}
	got := out.String()
	aIdx := strings.Index(got, `"a"`)
	mIdx := strings.Index(got, `"m"`)
	zIdx := strings.Index(got, `"z"`)
	if aIdx < 0 || mIdx < 0 || zIdx < 0 {
		t.Fatalf("Write() output = %q, missing expected keys", got)
	}
	if !(aIdx < mIdx && mIdx < zIdx) {
		t.Fatalf("Write() keys not sorted: %q", got)
	}
}

func TestWriteNestedObjectKeysSorted(t *testing.T) {
	input := `{"outer": {"z": 1, "a": 2}}`
	var out strings.Builder
	if err := Write(&out, strings.NewReader(input), Options{}); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}
	got := out.String()
	aIdx := strings.Index(got, `"a"`)
	zIdx := strings.Index(got, `"z"`)
	if aIdx < 0 || zIdx < 0 {
		t.Fatalf("Write() output = %q, missing nested keys", got)
	}
	if aIdx >= zIdx {
		t.Fatalf("Write() nested keys not sorted: %q", got)
	}
}

func TestWriteCompact(t *testing.T) {
	var out strings.Builder
	if err := Write(&out, strings.NewReader(`{"b": 2, "a": 1}`), Options{Compact: true}); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}
	got := strings.TrimSpace(out.String())
	if strings.Contains(got, "\n") {
		t.Fatalf("Write() compact output contains newlines: %q", got)
	}
	if got != `{"a":1,"b":2}` {
		t.Fatalf("Write() output = %q, want %q", got, `{"a":1,"b":2}`)
	}
}

func TestWritePreservesArrayOrder(t *testing.T) {
	var out strings.Builder
	if err := Write(&out, strings.NewReader(`{"arr": [3, 1, 2]}`), Options{Compact: true}); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != `{"arr":[3,1,2]}` {
		t.Fatalf("Write() output = %q, want %q", got, `{"arr":[3,1,2]}`)
	}
}

func TestWriteSortArrays(t *testing.T) {
	var out strings.Builder
	if err := Write(&out, strings.NewReader(`{"arr": [3, 1, 2]}`), Options{SortArrays: true, Compact: true}); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != `{"arr":[1,2,3]}` {
		t.Fatalf("Write() output = %q, want %q", got, `{"arr":[1,2,3]}`)
	}
}

func TestWriteSortArraysNestedScalarArrays(t *testing.T) {
	var out strings.Builder
	if err := Write(&out, strings.NewReader(`{"outer":{"arr":["b","a"]}}`), Options{SortArrays: true, Compact: true}); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != `{"outer":{"arr":["a","b"]}}` {
		t.Fatalf("Write() output = %q, want %q", got, `{"outer":{"arr":["a","b"]}}`)
	}
}

func TestWriteSortArraysPreservesObjectArrayOrder(t *testing.T) {
	var out strings.Builder
	if err := Write(&out, strings.NewReader(`{"arr": [{"b": 2}, {"a": 1}]}`), Options{SortArrays: true, Compact: true}); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}
	got := strings.TrimSpace(out.String())
	bIdx := strings.Index(got, `"b"`)
	aIdx := strings.Index(got, `"a"`)
	if bIdx < 0 || aIdx < 0 {
		t.Fatalf("Write() output = %q, missing expected keys", got)
	}
	if bIdx >= aIdx {
		t.Fatalf("Write() object-array order not preserved: %q", got)
	}
}

func TestWriteInvalidJSON(t *testing.T) {
	err := Write(&strings.Builder{}, strings.NewReader(`{invalid}`), Options{})
	if err == nil {
		t.Fatal("Write() expected error for invalid JSON")
	}
}

func TestCompareScalars(t *testing.T) {
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

func TestWriteSortArraysMixedScalarTypes(t *testing.T) {
	var out strings.Builder
	if err := Write(&out, strings.NewReader(`{"arr":["b",null,false,2,"a"]}`), Options{SortArrays: true, Compact: true}); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != `{"arr":[null,false,2,"a","b"]}` {
		t.Fatalf("Write() output = %q, want %q", got, `{"arr":[null,false,2,"a","b"]}`)
	}
}

func TestWriteSortArraysNumericOrder(t *testing.T) {
	var out strings.Builder
	if err := Write(&out, strings.NewReader(`{"arr":[2,10,1]}`), Options{SortArrays: true, Compact: true}); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != `{"arr":[1,2,10]}` {
		t.Fatalf("Write() output = %q, want %q", got, `{"arr":[1,2,10]}`)
	}
}

func TestWritePreservesLargeInteger(t *testing.T) {
	var out strings.Builder
	if err := Write(&out, strings.NewReader(`{"n":9007199254740993}`), Options{Compact: true}); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != `{"n":9007199254740993}` {
		t.Fatalf("Write() output = %q, want %q", got, `{"n":9007199254740993}`)
	}
}

func TestWriteRejectsTrailingContent(t *testing.T) {
	err := Write(&strings.Builder{}, strings.NewReader(`{"a":1} trailing`), Options{})
	if err == nil {
		t.Fatal("Write() expected error for trailing content")
	}
	if !strings.Contains(err.Error(), "extra content") && !strings.Contains(err.Error(), "invalid character") {
		t.Fatalf("Write() error = %q, want trailing-content parse error", err)
	}
}
