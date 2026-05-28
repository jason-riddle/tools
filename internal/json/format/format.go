// Package format provides stable JSON formatting with sorted object keys.
//
// Array order is preserved by default. When requested, arrays containing only
// scalar values are sorted recursively. Object key sorting can be scoped to a
// specific depth range, preserving original key order outside that range.
package format

import (
	"bytes"
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
	// SortArrays enables recursive sorting of arrays of scalar values.
	SortArrays bool
	// ArraysDepth limits how many array levels deep SortArrays recurses.
	// -1 (default) means unlimited. Positive values decrement on each array
	// boundary crossed; sorting stops when the counter reaches 0.
	// Object traversal does not consume depth.
	ArraysDepth int
	// SortKeysMinDepth is the first object level at which key sorting begins.
	// Default 1 means sorting starts at the top-level object.
	// Set to 2 to leave top-level keys in input order and start sorting one
	// level down. Object traversal increments the level; arrays do not.
	SortKeysMinDepth int
	// SortKeysMaxDepth is the last object level at which key sorting applies.
	// -1 (default) means unlimited — sort from SortKeysMinDepth downward.
	// Set equal to SortKeysMinDepth to sort exactly one object level.
	SortKeysMaxDepth int
	// Compact emits compact JSON instead of pretty-printed output.
	Compact bool
}

// Write reads a single JSON value from r, normalizes it, and writes it to w.
//
// It rejects trailing non-whitespace content after the first JSON value.
func Write(w io.Writer, r io.Reader, opts Options) error {
	// Normalise zero values: min<=1 means "start at top", max<=0 means "no upper bound".
	keysMin := opts.SortKeysMinDepth
	if keysMin < 1 {
		keysMin = 1
	}
	keysMax := opts.SortKeysMaxDepth
	if keysMax <= 0 {
		keysMax = -1
	}

	// Fast path: when sorting applies from depth 1 with no upper bound,
	// encoding/json handles key sorting automatically for every level.
	if keysMin == 1 && keysMax == -1 {
		return writeFast(w, r, opts)
	}
	return writeCustom(w, r, opts)
}

// writeFast is the original code path used when key sorting is unlimited
// from the top level down.
func writeFast(w io.Writer, r io.Reader, opts Options) error {
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
		ad := opts.ArraysDepth
		if ad == 0 {
			ad = -1
		}
		v = normalizeArrays(v, ad)
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

// --------------------------------------------------------------------------
// Ordered-node tree — used when key sorting is depth-range limited.
// --------------------------------------------------------------------------

// node is a parsed JSON value that preserves object key insertion order.
type node struct {
	// Exactly one of the following is set depending on the JSON type.
	raw      json.RawMessage // scalar: null, bool, number, string
	objKeys  []string        // object: key order
	objVals  map[string]*node
	arrElems []*node // array
}

// parseNode decodes raw JSON into an ordered node tree.
func parseNode(data json.RawMessage) (*node, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("empty JSON value")
	}
	switch data[0] {
	case '{':
		dec := json.NewDecoder(bytes.NewReader(data))
		if _, err := dec.Token(); err != nil { // consume '{'
			return nil, err
		}
		n := &node{objVals: make(map[string]*node)}
		for dec.More() {
			t, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key, ok := t.(string)
			if !ok {
				return nil, fmt.Errorf("expected string key, got %T", t)
			}
			var raw json.RawMessage
			if err := dec.Decode(&raw); err != nil {
				return nil, err
			}
			child, err := parseNode(raw)
			if err != nil {
				return nil, err
			}
			n.objKeys = append(n.objKeys, key)
			n.objVals[key] = child
		}
		if _, err := dec.Token(); err != nil { // consume '}'
			return nil, err
		}
		return n, nil

	case '[':
		dec := json.NewDecoder(bytes.NewReader(data))
		if _, err := dec.Token(); err != nil { // consume '['
			return nil, err
		}
		n := &node{}
		for dec.More() {
			var raw json.RawMessage
			if err := dec.Decode(&raw); err != nil {
				return nil, err
			}
			child, err := parseNode(raw)
			if err != nil {
				return nil, err
			}
			n.arrElems = append(n.arrElems, child)
		}
		if _, err := dec.Token(); err != nil { // consume ']'
			return nil, err
		}
		return n, nil

	default:
		return &node{raw: data}, nil
	}
}

// marshalNode writes n to buf.
//
// objDepth is the current object nesting level (1 = top-level object).
// Keys are sorted when objDepth is within [keysMin, keysMax] (keysMax=-1 means
// no upper bound). Array traversal does not change objDepth.
func marshalNode(buf *bytes.Buffer, n *node, indent string, compact bool, keysMin, keysMax, objDepth int, sortArrays bool, arraysDepth int) error {
	switch {
	case n.raw != nil:
		buf.Write(n.raw)

	case n.objVals != nil:
		keys := make([]string, len(n.objKeys))
		copy(keys, n.objKeys)
		shouldSort := objDepth >= keysMin && (keysMax == -1 || objDepth <= keysMax)
		if shouldSort {
			sort.Strings(keys)
		}

		buf.WriteByte('{')
		childIndent := indent + "  "
		for i, k := range keys {
			if !compact {
				buf.WriteByte('\n')
				buf.WriteString(childIndent)
			}
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteByte(':')
			if !compact {
				buf.WriteByte(' ')
			}
			if err := marshalNode(buf, n.objVals[k], childIndent, compact, keysMin, keysMax, objDepth+1, sortArrays, arraysDepth); err != nil {
				return err
			}
			if i < len(keys)-1 {
				buf.WriteByte(',')
			}
		}
		if !compact && len(keys) > 0 {
			buf.WriteByte('\n')
			buf.WriteString(indent)
		}
		buf.WriteByte('}')

	default: // array — does not consume objDepth
		elems := n.arrElems

		nextArraysDepth := arraysDepth
		if sortArrays && arraysDepth != 0 {
			if nextArraysDepth > 0 {
				nextArraysDepth--
			}
			allScalar := true
			for _, e := range elems {
				if e.raw == nil {
					allScalar = false
					break
				}
			}
			if allScalar && len(elems) > 0 {
				sort.Slice(elems, func(i, j int) bool {
					return compareScalars(rawToScalar(elems[i].raw), rawToScalar(elems[j].raw)) < 0
				})
			}
		} else {
			nextArraysDepth = arraysDepth // no change when not sorting
		}

		buf.WriteByte('[')
		childIndent := indent + "  "
		for i, e := range elems {
			if !compact {
				buf.WriteByte('\n')
				buf.WriteString(childIndent)
			}
			if err := marshalNode(buf, e, childIndent, compact, keysMin, keysMax, objDepth, sortArrays, nextArraysDepth); err != nil {
				return err
			}
			if i < len(elems)-1 {
				buf.WriteByte(',')
			}
		}
		if !compact && len(elems) > 0 {
			buf.WriteByte('\n')
			buf.WriteString(indent)
		}
		buf.WriteByte(']')
	}
	return nil
}

// rawToScalar converts a raw JSON scalar token into the same types that
// json.Decoder produces when using UseNumber, so compareScalars works.
func rawToScalar(raw json.RawMessage) any {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil
	}
	switch {
	case bytes.Equal(raw, []byte("null")):
		return nil
	case bytes.Equal(raw, []byte("true")):
		return true
	case bytes.Equal(raw, []byte("false")):
		return false
	case raw[0] == '"':
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s
		}
	default:
		return json.Number(string(raw))
	}
	return string(raw)
}

// writeCustom is the custom-marshaling path used when key sorting is
// depth-range limited.
func writeCustom(w io.Writer, r io.Reader, opts Options) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return fmt.Errorf("parse json: %w", err)
	}
	if err := dec.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("parse json: extra content after first JSON value")
		}
		return fmt.Errorf("parse json: %w", err)
	}

	root, err := parseNode(raw)
	if err != nil {
		return fmt.Errorf("parse json: %w", err)
	}

	ad := opts.ArraysDepth
	if ad == 0 {
		ad = -1
	}
	keysMin := opts.SortKeysMinDepth
	if keysMin < 1 {
		keysMin = 1
	}
	keysMax := opts.SortKeysMaxDepth
	if keysMax <= 0 {
		keysMax = -1
	}

	var buf bytes.Buffer
	if err := marshalNode(&buf, root, "", opts.Compact, keysMin, keysMax, 1, opts.SortArrays, ad); err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	buf.WriteByte('\n')
	_, err = w.Write(buf.Bytes())
	return err
}

// --------------------------------------------------------------------------
// Fast-path helpers (used by writeFast / normalizeArrays).
// --------------------------------------------------------------------------

// normalizeArrays recursively sorts scalar arrays in place.
// depth controls how many array levels deep to recurse: -1 means unlimited,
// positive values decrement on each array level and stop when they reach 0.
// Traversing into object values does not consume depth.
func normalizeArrays(v any, depth int) any {
	switch val := v.(type) {
	case map[string]any:
		for k, elem := range val {
			val[k] = normalizeArrays(elem, depth)
		}
		return val

	case []any:
		if depth != 0 {
			next := depth
			if next > 0 {
				next--
			}
			for i, elem := range val {
				val[i] = normalizeArrays(elem, next)
			}
			if isScalarSlice(val) {
				sort.Slice(val, func(i, j int) bool {
					return compareScalars(val[i], val[j]) < 0
				})
			}
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
