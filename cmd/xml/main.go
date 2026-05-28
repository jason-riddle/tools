package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	xmlformat "github.com/jason-riddle/tools/internal/xml/format"
)

const usage = `xml - pretty-print an XML file

Usage:
  xml [flags] [file]

Arguments:
  file  Optional path to an XML file. Reads from stdin when omitted.

Flags:
  -h, -help, --help
        Show help

Examples:
  xml < file.xml
  xml file.xml
`

var errUsage = errors.New("usage")

func main() {
	log.SetFlags(0)
	log.SetPrefix("xml: ")

	if err := run(os.Args[1:]); err != nil {
		if !errors.Is(err, errUsage) {
			log.Print(err)
		}
		os.Exit(1)
	}
}

func run(args []string) error {
	file, err := parseOptions(args)
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

	return xmlformat.Write(os.Stdout, r)
}

func printUsageError(err error) {
	fmt.Fprintf(os.Stderr, "xml: %v\n\n%s", err, usage)
}

func parseOptions(args []string) (string, error) {
	fs := flag.NewFlagSet("xml", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { fmt.Fprint(os.Stdout, usage) }

	if err := fs.Parse(args); err != nil {
		return "", err
	}

	switch fs.NArg() {
	case 0:
		return "", nil
	case 1:
		return fs.Arg(0), nil
	default:
		return "", errors.New("accepts at most one file argument")
	}
}
