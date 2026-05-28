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

func TestWriteSortArraysUnlimitedDepth(t *testing.T) {
	// Nested scalar arrays should all be sorted when depth is -1 (unlimited).
	// The inner slices [5,4] and [8,6] get sorted; the outer [][]any is not
	// a scalar slice and keeps its order.
	input := `{"a":[3,1,2]}`
	out := mustWrite(t, input, format.Options{SortArrays: true, Depth: -1, Compact: true})
	if !strings.Contains(out, `"a":[1,2,3]`) {
		t.Errorf("got %s, want a sorted to [1,2,3]", out)
	}
}

func TestWriteSortArraysDepth1(t *testing.T) {
	// depth=1: arrays at the first array-level encountered are sorted.
	// Object traversal does not consume depth, so arrays nested inside
	// objects at the same array-depth are also sorted.
	// Arrays nested inside other arrays are NOT sorted.
	input := `{"top":[3,1,2],"matrix":[[5,4],[8,6]]}`
	out := mustWrite(t, input, format.Options{SortArrays: true, Depth: 1, Compact: true})
	if !strings.Contains(out, `"top":[1,2,3]`) {
		t.Errorf("got %s, want top sorted to [1,2,3]", out)
	}
	// matrix is not a scalar slice so it won't be sorted, but its inner
	// slices are at array-depth 2 and should NOT be sorted with depth=1.
	if strings.Contains(out, `[4,5]`) || strings.Contains(out, `[6,8]`) {
		t.Errorf("got %s, inner arrays should not be sorted with depth=1", out)
	}
}

func TestWriteSortArraysDepth0MeansUnlimited(t *testing.T) {
	// depth=0 is treated the same as -1 (unlimited).
	input := `{"a":[3,1,2]}`
	out := mustWrite(t, input, format.Options{SortArrays: true, Depth: 0, Compact: true})
	if !strings.Contains(out, `"a":[1,2,3]`) {
		t.Errorf("got %s, want a sorted to [1,2,3]", out)
	}
}

func TestWriteSortArraysDisabled(t *testing.T) {
	input := `{"a":[3,1,2]}`
	out := mustWrite(t, input, format.Options{SortArrays: false, Depth: -1, Compact: true})
	if !strings.Contains(out, `"a":[3,1,2]`) {
		t.Errorf("got %s, want a left as [3,1,2]", out)
	}
}
