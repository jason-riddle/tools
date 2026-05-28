package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"sort"
	"strconv"
)

const usage = `json - pretty-print JSON with sorted keys

Usage:
  json [flags] [file]

Arguments:
  file  Optional path to a JSON file. Reads from stdin when omitted.

Flags:
  --sort-arrays
        Sort arrays of scalar values recursively; arrays containing objects
        or nested arrays keep their original order.
  --compact
        Emit compact JSON instead of pretty-printed output.
  -h, -help, --help
        Show help

Examples:
  json < file.json
  json file.json
  json --compact file.json
  json --sort-arrays file.json
`

var errUsage = errors.New("usage")

type options struct {
	sortArrays bool
	compact    bool
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("json: ")

	if err := run(os.Args[1:]); err != nil {
		if !errors.Is(err, errUsage) {
			log.Print(err)
		}
		os.Exit(1)
	}
}

func run(args []string) error {
	opts, file, err := parseOptions(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		printUsageError(err)
		return errUsage
	}

	var r io.Reader
	if file == "" {
		r = os.Stdin
	} else {
		f, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("open %s: %w", file, err)
		}
		defer f.Close()
		r = f
	}

	return writeJSON(os.Stdout, r, opts)
}

func printUsageError(err error) {
	fmt.Fprintf(os.Stderr, "json: %v\n\n%s", err, usage)
}

func parseOptions(args []string) (options, string, error) {
	var opts options

	fs := flag.NewFlagSet("json", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { fmt.Fprint(os.Stdout, usage) }
	fs.BoolVar(&opts.sortArrays, "sort-arrays", false, "sort arrays of scalar values recursively")
	fs.BoolVar(&opts.compact, "compact", false, "emit compact JSON instead of pretty-printed output")

	if err := fs.Parse(args); err != nil {
		return options{}, "", err
	}

	switch fs.NArg() {
	case 0:
		return opts, "", nil
	case 1:
		return opts, fs.Arg(0), nil
	default:
		return options{}, "", errors.New("accepts at most one file argument")
	}
}

func writeJSON(w io.Writer, r io.Reader, opts options) error {
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

	if opts.sortArrays {
		v = normalizeArrays(v)
	}

	enc := json.NewEncoder(w)
	if !opts.compact {
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
