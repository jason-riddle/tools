// Package format provides stable JSON formatting with sorted object keys.
//
// Array order is preserved by default. When requested, arrays containing only
// scalar values are sorted recursively.
package format

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"
	"strconv"
)

// Options configures JSON normalization and output formatting.
type Options struct {
	SortArrays bool
	Compact    bool
}

// Write reads a single JSON value from r, normalizes it, and writes it to w.
//
// It rejects trailing non-whitespace content after the first JSON value.
func Write(w io.Writer, r io.Reader, opts Options) error {
	var v any
	dec := json.NewDecoder(r)
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return fmt.Errorf("parse json: %w", err)
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("parse json: extra content after first JSON value")
		}
		return fmt.Errorf("parse json: %w", err)
	}

	if opts.SortArrays {
		v = normalizeArrays(v)
	}

	enc := json.NewEncoder(w)
	if !opts.Compact {
		enc.SetIndent("", "  ")
	}

	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	return nil
}

// normalizeArrays recursively sorts scalar arrays in place.
func normalizeArrays(v any) any {
	switch val := v.(type) {
	case map[string]any:
		for k, elem := range val {
			val[k] = normalizeArrays(elem)
		}
		return val

	case []any:
		for i, elem := range val {
			val[i] = normalizeArrays(elem)
		}
		if isScalarSlice(val) {
			sort.Slice(val, func(i, j int) bool {
				return compareScalars(val[i], val[j]) < 0
			})
		}
		return val

	default:
		return v
	}
}

// isScalarSlice reports whether every element in s is a scalar (not a map or slice).
func isScalarSlice(s []any) bool {
	for _, elem := range s {
		switch elem.(type) {
		case map[string]any, []any:
			return false
		}
	}
	return true
}

func compareScalars(a, b any) int {
	ra, rb := scalarRank(a), scalarRank(b)
	if ra != rb {
		if ra < rb {
			return -1
		}
		return 1
	}

	switch av := a.(type) {
	case nil:
		return 0
	case bool:
		bv := b.(bool)
		switch {
		case !av && bv:
			return -1
		case av && !bv:
			return 1
		default:
			return 0
		}
	case json.Number:
		return compareNumbers(av.String(), b.(json.Number).String())
	case float64:
		return compareNumbers(strconv.FormatFloat(av, 'g', -1, 64), strconv.FormatFloat(b.(float64), 'g', -1, 64))
	case string:
		return stringsCompare(av, b.(string))
	default:
		return stringsCompare(fmt.Sprintf("%T:%v", a, a), fmt.Sprintf("%T:%v", b, b))
	}
}

func scalarRank(v any) int {
	switch v.(type) {
	case nil:
		return 0
	case bool:
		return 1
	case json.Number, float64:
		return 2
	case string:
		return 3
	default:
		return 4
	}
}

func compareNumbers(a, b string) int {
	af, ok := new(big.Float).SetString(a)
	if !ok {
		return stringsCompare(a, b)
	}
	bf, ok := new(big.Float).SetString(b)
	if !ok {
		return stringsCompare(a, b)
	}
	if cmp := af.Cmp(bf); cmp != 0 {
		return cmp
	}
	return stringsCompare(a, b)
}

func stringsCompare(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
