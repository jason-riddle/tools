package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	jsonformat "github.com/jason-riddle/tools/internal/json/format"
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
  --depth N
        Limit recursive array sorting to N levels deep. Requires --sort-arrays.
        -1 (default) means unlimited; 1 sorts only the top-level array.
  --compact
        Emit compact JSON instead of pretty-printed output.
  -h, -help, --help
        Show help

Examples:
  json < file.json
  json file.json
  json --compact file.json
  json --sort-arrays file.json
  json --sort-arrays --depth 1 file.json
`

var errUsage = errors.New("usage")

type options struct {
	sortArrays bool
	compact    bool
	depth      int
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

	return jsonformat.Write(os.Stdout, r, jsonformat.Options{
		SortArrays: opts.sortArrays,
		Compact:    opts.compact,
		Depth:      opts.depth,
	})
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
	fs.IntVar(&opts.depth, "depth", -1, "limit array sorting recursion depth (-1 means unlimited)")

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
