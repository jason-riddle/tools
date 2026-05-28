package format_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jason-riddle/tools/internal/json/format"
)

// pluginRegistry is a generalised form of a lock file pattern: a top-level
// version scalar plus a registry object whose keys are entry names and whose
// values are metadata objects with fixed fields. Tests use this shape
// throughout to stay grounded in a realistic use case.
//
//	{
//	  "version": 3,
//	  "plugins": {
//	    "router": {"source": "org/plugins", "installedAt": "2026-01-01"},
//	    "auth":   {"source": "org/plugins", "installedAt": "2026-02-01"}
//	  }
//	}
const pluginRegistry = `{"version":3,"plugins":{"router":{"source":"org/plugins","installedAt":"2026-01-01"},"auth":{"source":"org/plugins","installedAt":"2026-02-01"}}}`

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
	// Tags inside a plugin entry should be sorted when arrays-depth is unlimited.
	input := `{"version":3,"plugins":{"auth":{"tags":["z","a","m"]}}}`
	out := mustWrite(t, input, format.Options{SortArrays: true, ArraysDepth: -1, Compact: true})
	if !strings.Contains(out, `"tags":["a","m","z"]`) {
		t.Errorf("got %s, want tags sorted to [a,m,z]", out)
	}
}

func TestWriteSortArraysDepth1(t *testing.T) {
	// arrays-depth=1: first array level is sorted; arrays nested inside arrays are not.
	input := `{"version":3,"plugins":{"auth":{"tags":["z","a","m"],"matrix":[[5,4],[8,6]]}}}`
	out := mustWrite(t, input, format.Options{SortArrays: true, ArraysDepth: 1, Compact: true})
	if !strings.Contains(out, `"tags":["a","m","z"]`) {
		t.Errorf("got %s, want tags sorted to [a,m,z]", out)
	}
	// Inner arrays inside matrix are at depth 2 — not sorted with arrays-depth=1.
	if strings.Contains(out, `[4,5]`) || strings.Contains(out, `[6,8]`) {
		t.Errorf("got %s, inner arrays should not be sorted with arrays-depth=1", out)
	}
}

func TestWriteSortArraysDepth0MeansUnlimited(t *testing.T) {
	// arrays-depth=0 is treated the same as -1 (unlimited).
	input := `{"version":3,"plugins":{"auth":{"tags":["z","a","m"]}}}`
	out := mustWrite(t, input, format.Options{SortArrays: true, ArraysDepth: 0, Compact: true})
	if !strings.Contains(out, `"tags":["a","m","z"]`) {
		t.Errorf("got %s, want tags sorted to [a,m,z]", out)
	}
}

func TestWriteSortArraysDisabled(t *testing.T) {
	input := `{"version":3,"plugins":{"auth":{"tags":["z","a","m"]}}}`
	out := mustWrite(t, input, format.Options{SortArrays: false, Compact: true})
	if !strings.Contains(out, `"tags":["z","a","m"]`) {
		t.Errorf("got %s, want tags left in input order [z,a,m]", out)
	}
}

// --------------------------------------------------------------------------
// SortKeysMinDepth / SortKeysMaxDepth tests
// --------------------------------------------------------------------------

func TestWriteSortKeysDefault(t *testing.T) {
	// Default (min=1, max=-1): all object keys sorted at every level.
	// Top-level: plugins before version.
	// Depth-2: auth before router.
	// Depth-3: installedAt before source (within each plugin).
	out := mustWrite(t, pluginRegistry, format.Options{Compact: true})

	pluginsIdx := strings.Index(out, `"plugins"`)
	versionIdx := strings.Index(out, `"version"`)
	if pluginsIdx < 0 || versionIdx < 0 || pluginsIdx >= versionIdx {
		t.Errorf("got %s, want plugins before version at top level", out)
	}

	authIdx := strings.Index(out, `"auth"`)
	routerIdx := strings.Index(out, `"router"`)
	if authIdx < 0 || routerIdx < 0 || authIdx >= routerIdx {
		t.Errorf("got %s, want auth before router (sorted plugin names)", out)
	}

	installedIdx := strings.Index(out, `"installedAt"`)
	sourceIdx := strings.Index(out, `"source"`)
	if installedIdx < 0 || sourceIdx < 0 || installedIdx >= sourceIdx {
		t.Errorf("got %s, want installedAt before source inside plugin (sorted)", out)
	}
}

func TestWriteSortKeysMinDepth2MaxDepth2(t *testing.T) {
	// min=2, max=2: leave top-level keys in input order; sort only plugin names.
	// This is the lock-file use case: version stays first, plugin names are
	// alphabetical, each plugin's inner fields keep their original input order.
	out := mustWrite(t, pluginRegistry, format.Options{SortKeysMinDepth: 2, SortKeysMaxDepth: 2, Compact: true})

	// Top-level: version before plugins (input order preserved).
	versionIdx := strings.Index(out, `"version"`)
	pluginsIdx := strings.Index(out, `"plugins"`)
	if versionIdx < 0 || pluginsIdx < 0 || versionIdx >= pluginsIdx {
		t.Errorf("got %s, want version before plugins (input order)", out)
	}

	// Plugin names: auth before router (sorted at depth 2).
	authIdx := strings.Index(out, `"auth"`)
	routerIdx := strings.Index(out, `"router"`)
	if authIdx < 0 || routerIdx < 0 || authIdx >= routerIdx {
		t.Errorf("got %s, want auth before router (sorted)", out)
	}

	// Inside each plugin: source before installedAt (input order preserved at depth 3).
	sourceIdx := strings.Index(out, `"source"`)
	installedIdx := strings.Index(out, `"installedAt"`)
	if sourceIdx < 0 || installedIdx < 0 || sourceIdx >= installedIdx {
		t.Errorf("got %s, want source before installedAt inside plugin (input order)", out)
	}
}

func TestWriteSortKeysMaxDepth1(t *testing.T) {
	// min=1, max=1: sort only top-level keys; everything below keeps input order.
	out := mustWrite(t, pluginRegistry, format.Options{SortKeysMinDepth: 1, SortKeysMaxDepth: 1, Compact: true})

	// Top-level: plugins before version (sorted).
	pluginsIdx := strings.Index(out, `"plugins"`)
	versionIdx := strings.Index(out, `"version"`)
	if pluginsIdx < 0 || versionIdx < 0 || pluginsIdx >= versionIdx {
		t.Errorf("got %s, want plugins before version (sorted at depth 1)", out)
	}

	// Plugin names: router before auth (input order preserved at depth 2).
	routerIdx := strings.Index(out, `"router"`)
	authIdx := strings.Index(out, `"auth"`)
	if routerIdx < 0 || authIdx < 0 || routerIdx >= authIdx {
		t.Errorf("got %s, want router before auth (input order at depth 2)", out)
	}
}

func TestWriteSortKeysMinDepth2Unlimited(t *testing.T) {
	// min=2, max=-1: top-level keys in input order; sort everything from depth 2 down.
	out := mustWrite(t, pluginRegistry, format.Options{SortKeysMinDepth: 2, SortKeysMaxDepth: -1, Compact: true})

	// Top-level: version before plugins (input order).
	versionIdx := strings.Index(out, `"version"`)
	pluginsIdx := strings.Index(out, `"plugins"`)
	if versionIdx < 0 || pluginsIdx < 0 || versionIdx >= pluginsIdx {
		t.Errorf("got %s, want version before plugins (input order at depth 1)", out)
	}

	// Plugin names: auth before router (sorted at depth 2).
	authIdx := strings.Index(out, `"auth"`)
	routerIdx := strings.Index(out, `"router"`)
	if authIdx < 0 || routerIdx < 0 || authIdx >= routerIdx {
		t.Errorf("got %s, want auth before router (sorted at depth 2)", out)
	}

	// Inside each plugin: installedAt before source (sorted at depth 3).
	installedIdx := strings.Index(out, `"installedAt"`)
	sourceIdx := strings.Index(out, `"source"`)
	if installedIdx < 0 || sourceIdx < 0 || installedIdx >= sourceIdx {
		t.Errorf("got %s, want installedAt before source inside plugin (sorted at depth 3)", out)
	}
}

func TestWriteSortKeysArrayPassthrough(t *testing.T) {
	// Arrays do not consume key-sort depth. An object inside an array counts
	// as one deeper than the enclosing object.
	// With min=2, max=2: the plugin objects inside the "items" array are at
	// depth 2, so their keys are sorted.
	input := `{"version":3,"items":[{"source":"org/plugins","installedAt":"2026-01-01"}]}`
	out := mustWrite(t, input, format.Options{SortKeysMinDepth: 2, SortKeysMaxDepth: 2, Compact: true})

	// Top-level: version before items (input order).
	versionIdx := strings.Index(out, `"version"`)
	itemsIdx := strings.Index(out, `"items"`)
	if versionIdx < 0 || itemsIdx < 0 || versionIdx >= itemsIdx {
		t.Errorf("got %s, want version before items (input order at depth 1)", out)
	}

	// Object inside array is at depth 2 — installedAt before source (sorted).
	installedIdx := strings.Index(out, `"installedAt"`)
	sourceIdx := strings.Index(out, `"source"`)
	if installedIdx < 0 || sourceIdx < 0 || installedIdx >= sourceIdx {
		t.Errorf("got %s, want installedAt before source in array object (sorted at depth 2)", out)
	}
}
