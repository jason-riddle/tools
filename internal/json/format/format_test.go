package format_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jason-riddle/tools/internal/json/format"
)

func mustWrite(t *testing.T, input string, opts format.Options) string {
	t.Helper()
	var buf bytes.Buffer
	if err := format.Write(&buf, strings.NewReader(input), opts); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	return buf.String()
}

// --------------------------------------------------------------------------
// SortArrays / ArraysDepth tests
// --------------------------------------------------------------------------

func TestWriteSortArraysUnlimitedDepth(t *testing.T) {
	input := `{"a":[3,1,2]}`
	out := mustWrite(t, input, format.Options{SortArrays: true, ArraysDepth: -1, Compact: true})
	if !strings.Contains(out, `"a":[1,2,3]`) {
		t.Errorf("got %s, want a sorted to [1,2,3]", out)
	}
}

func TestWriteSortArraysDepth1(t *testing.T) {
	// arrays-depth=1: first array level is sorted; arrays nested inside arrays are not.
	input := `{"top":[3,1,2],"matrix":[[5,4],[8,6]]}`
	out := mustWrite(t, input, format.Options{SortArrays: true, ArraysDepth: 1, Compact: true})
	if !strings.Contains(out, `"top":[1,2,3]`) {
		t.Errorf("got %s, want top sorted to [1,2,3]", out)
	}
	// matrix outer is not scalar so not sorted; inner arrays are at depth 2 — not sorted.
	if strings.Contains(out, `[4,5]`) || strings.Contains(out, `[6,8]`) {
		t.Errorf("got %s, inner arrays should not be sorted with arrays-depth=1", out)
	}
}

func TestWriteSortArraysDepth0MeansUnlimited(t *testing.T) {
	// arrays-depth=0 is treated the same as -1 (unlimited).
	input := `{"a":[3,1,2]}`
	out := mustWrite(t, input, format.Options{SortArrays: true, ArraysDepth: 0, Compact: true})
	if !strings.Contains(out, `"a":[1,2,3]`) {
		t.Errorf("got %s, want a sorted to [1,2,3]", out)
	}
}

func TestWriteSortArraysDisabled(t *testing.T) {
	input := `{"a":[3,1,2]}`
	out := mustWrite(t, input, format.Options{SortArrays: false, Compact: true})
	if !strings.Contains(out, `"a":[3,1,2]`) {
		t.Errorf("got %s, want a left as [3,1,2]", out)
	}
}

// --------------------------------------------------------------------------
// SortKeysMinDepth / SortKeysMaxDepth tests
// --------------------------------------------------------------------------

func TestWriteSortKeysDefault(t *testing.T) {
	// Default (min=1, max=-1): all object keys sorted at every level.
	input := `{"z":{"y":1,"x":2},"b":{"d":3,"c":4}}`
	out := mustWrite(t, input, format.Options{Compact: true})
	// Top-level: b before z.
	bIdx := strings.Index(out, `"b"`)
	zIdx := strings.Index(out, `"z"`)
	if bIdx < 0 || zIdx < 0 || bIdx >= zIdx {
		t.Errorf("got %s, want top-level keys sorted (b before z)", out)
	}
	// Nested: c before d, x before y.
	cIdx := strings.Index(out, `"c"`)
	dIdx := strings.Index(out, `"d"`)
	if cIdx < 0 || dIdx < 0 || cIdx >= dIdx {
		t.Errorf("got %s, want nested keys sorted (c before d)", out)
	}
}

func TestWriteSortKeysMaxDepth1(t *testing.T) {
	// min=1, max=1: sort only top-level keys; nested keys keep input order.
	input := `{"z":{"y":1,"x":2},"b":{"d":3,"c":4}}`
	out := mustWrite(t, input, format.Options{SortKeysMinDepth: 1, SortKeysMaxDepth: 1, Compact: true})

	// Top-level: b before z.
	bIdx := strings.Index(out, `"b"`)
	zIdx := strings.Index(out, `"z"`)
	if bIdx < 0 || zIdx < 0 || bIdx >= zIdx {
		t.Errorf("got %s, want top-level keys sorted (b before z)", out)
	}
	// Inside "b": d before c (input order).
	dIdx := strings.Index(out, `"d"`)
	cIdx := strings.Index(out, `"c"`)
	if dIdx < 0 || cIdx < 0 || dIdx >= cIdx {
		t.Errorf("got %s, want nested keys in input order (d before c)", out)
	}
	// Inside "z": y before x (input order).
	yIdx := strings.Index(out, `"y"`)
	xIdx := strings.Index(out, `"x"`)
	if yIdx < 0 || xIdx < 0 || yIdx >= xIdx {
		t.Errorf("got %s, want nested keys in input order (y before x)", out)
	}
}

func TestWriteSortKeysMinDepth2MaxDepth2(t *testing.T) {
	// min=2, max=2: leave top-level keys in input order; sort only depth-2 keys.
	// This is the skill-lock.json use case: preserve version/skills order but
	// sort skill names inside the "skills" object.
	input := `{"version":3,"skills":{"z-skill":{},"a-skill":{}}}`
	out := mustWrite(t, input, format.Options{SortKeysMinDepth: 2, SortKeysMaxDepth: 2, Compact: true})

	// Top-level: version before skills (input order preserved).
	vIdx := strings.Index(out, `"version"`)
	sIdx := strings.Index(out, `"skills"`)
	if vIdx < 0 || sIdx < 0 || vIdx >= sIdx {
		t.Errorf("got %s, want version before skills (input order)", out)
	}
	// Inside "skills": a-skill before z-skill (sorted).
	aIdx := strings.Index(out, `"a-skill"`)
	zIdx := strings.Index(out, `"z-skill"`)
	if aIdx < 0 || zIdx < 0 || aIdx >= zIdx {
		t.Errorf("got %s, want a-skill before z-skill (sorted)", out)
	}
}

func TestWriteSortKeysMinDepth2Unlimited(t *testing.T) {
	// min=2, max=-1: top-level keys in input order; sort everything from depth 2 down.
	input := `{"z":{"d":{"b":1,"a":2},"c":3},"a":{}}`
	out := mustWrite(t, input, format.Options{SortKeysMinDepth: 2, SortKeysMaxDepth: -1, Compact: true})

	// Top-level: z before a (input order preserved).
	zIdx := strings.Index(out, `"z"`)
	aIdx := strings.Index(out, `"a"`)
	if zIdx < 0 || aIdx < 0 || zIdx >= aIdx {
		t.Errorf("got %s, want z before a at top level (input order)", out)
	}
	// Inside "z": c before d (sorted at depth 2).
	cIdx := strings.Index(out, `"c"`)
	dIdx := strings.Index(out, `"d"`)
	if cIdx < 0 || dIdx < 0 || cIdx >= dIdx {
		t.Errorf("got %s, want c before d inside z (sorted at depth 2)", out)
	}
}

func TestWriteSortKeysArrayPassthrough(t *testing.T) {
	// Arrays do not consume key-sort depth; objects inside arrays count as
	// one deeper than the array's enclosing object.
	// min=2, max=2: object inside the array is at depth 2 — its keys are sorted.
	input := `{"z":[{"b":2,"a":1}]}`
	out := mustWrite(t, input, format.Options{SortKeysMinDepth: 2, SortKeysMaxDepth: 2, Compact: true})

	// Top-level: z in input order (only one key, trivially fine).
	// Object inside array is at depth 2 — a before b (sorted).
	aIdx := strings.Index(out, `"a"`)
	bIdx := strings.Index(out, `"b"`)
	if aIdx < 0 || bIdx < 0 || aIdx >= bIdx {
		t.Errorf("got %s, want object-in-array keys sorted (a before b) at depth 2", out)
	}
}
