package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/jason-riddle/tools/internal/uuid"
)

const usage = `uuid - generate and inspect UUIDs

Usage:
  uuid <command> [flags]

Commands:
  new       Generate a new UUID
  parse     Parse a UUID string and print details
  version   Print the version number of a UUID string

Run 'uuid <command> -h' for command-specific help.

Global help:
  -h, -help, --help
        Show help

Examples:
  uuid new
  uuid new -v 7
  uuid parse f81d4fae-7dec-11d0-a765-00a0c91e6bf6
  uuid version f81d4fae-7dec-11d0-a765-00a0c91e6bf6
`

var errUsage = errors.New("usage")

func main() {
	log.SetFlags(0)
	log.SetPrefix("uuid: ")

	if err := run(os.Args[1:]); err != nil {
		if !errors.Is(err, errUsage) {
			log.Print(err)
		}
		os.Exit(1)
	}

}

func run(args []string) error {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, usage)
		return errUsage
	}

	switch args[0] {
	case "new":
		return runNew(args[1:])
	case "parse":
		return runParse(args[1:])
	case "version":
		return runVersion(args[1:])
	case "-h", "-help", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
		return nil
	default:
		fmt.Fprintf(os.Stderr, "uuid: unknown command %q\n\n", args[0])
		fmt.Fprint(os.Stderr, usage)
		return errUsage
	}
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "-h", "-help", "--help":
			return true
		}
	}

	return false
}

// runNew generates a new UUID of the requested version.
func runNew(args []string) error {
	if wantsHelp(args) {
		printNewUsage(os.Stdout)
		return nil
	}

	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	ver := fs.Int("v", 4, "UUID version to generate: 4 or 7")
	fs.Usage = func() {
		printNewUsage(os.Stderr)
	}
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "uuid new: %v\n\n", err)
		fs.Usage()
		return errUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "uuid new: unexpected arguments: %s\n\n", strings.Join(fs.Args(), " "))
		fs.Usage()
		return errUsage
	}

	var u uuid.UUID
	switch *ver {
	case 4:
		u = uuid.NewV4()
	case 7:
		u = uuid.NewV7()
	default:
		return fmt.Errorf("unsupported version %d; use 4 or 7", *ver)
	}
	fmt.Println(u)

	return nil
}

// runParse parses a UUID string and prints structured details.
func runParse(args []string) error {
	if wantsHelp(args) {
		printParseUsage(os.Stdout)
		return nil
	}

	fs := flag.NewFlagSet("parse", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		printParseUsage(os.Stderr)
	}
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "uuid parse: %v\n\n", err)
		fs.Usage()
		return errUsage
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "uuid parse: expected exactly one uuid-string argument")
		fmt.Fprintln(os.Stderr)
		fs.Usage()
		return errUsage
	}

	u, err := uuid.Parse(fs.Arg(0))
	if err != nil {
		return err
	}

	fmt.Printf("uuid:    %s\n", u)
	fmt.Printf("version: %d\n", u.Version())
	fmt.Printf("variant: %s\n", u.Variant())
	fmt.Printf("nil:     %v\n", u.IsNil())
	fmt.Printf("max:     %v\n", u.IsMax())

	return nil
}

// runVersion prints only the version number of a UUID string.
func runVersion(args []string) error {
	if wantsHelp(args) {
		printVersionUsage(os.Stdout)
		return nil
	}

	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		printVersionUsage(os.Stderr)
	}
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "uuid version: %v\n\n", err)
		fs.Usage()
		return errUsage
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "uuid version: expected exactly one uuid-string argument")
		fmt.Fprintln(os.Stderr)
		fs.Usage()
		return errUsage
	}

	u, err := uuid.Parse(fs.Arg(0))
	if err != nil {
		return err
	}

	fmt.Printf("%d\n", u.Version())

	return nil
}

func printNewUsage(w io.Writer) {
	fmt.Fprint(w, `uuid new - generate a new UUID

Usage:
  uuid new [flags]

Flags:
  -v int
        UUID version to generate: 4 or 7 (default 4)

Examples:
  uuid new
  uuid new -v 7
`)
}

func printParseUsage(w io.Writer) {
	fmt.Fprint(w, `uuid parse - parse a UUID string and print details

Usage:
  uuid parse <uuid-string>

Arguments:
  uuid-string  UUID to parse

Examples:
  uuid parse f81d4fae-7dec-11d0-a765-00a0c91e6bf6
`)
}

func printVersionUsage(w io.Writer) {
	fmt.Fprint(w, `uuid version - print the version number of a UUID string

Usage:
  uuid version <uuid-string>

Arguments:
  uuid-string  UUID to inspect

Examples:
  uuid version f81d4fae-7dec-11d0-a765-00a0c91e6bf6
`)
}
