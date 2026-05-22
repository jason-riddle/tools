package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jason-riddle/tools/internal/uuid"
)

const usage = `uuid - generate and inspect UUIDs

Usage:
  uuid <command> [flags]

Commands:
  new       Generate a new UUID
  parse     Parse a UUID string and print details
  version   Print the version number of a UUID string

New flags:
  -v int   UUID version to generate: 4 or 7 (default 4)

Parse flags:
  (none)   uuid parse <uuid-string>

Version flags:
  (none)   uuid version <uuid-string>

Examples:
  uuid new
  uuid new -v 7
  uuid parse f81d4fae-7dec-11d0-a765-00a0c91e6bf6
  uuid version f81d4fae-7dec-11d0-a765-00a0c91e6bf6
`

func main() {
	log.SetFlags(0)
	log.SetPrefix("uuid: ")

	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new":
		runNew(os.Args[2:])
	case "parse":
		runParse(os.Args[2:])
	case "version":
		runVersion(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
	default:
		fmt.Fprintf(os.Stderr, "uuid: unknown command %q\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
}

// runNew generates a new UUID of the requested version.
func runNew(args []string) {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	ver := fs.Int("v", 4, "UUID version to generate: 4 or 7")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: uuid new [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	var u uuid.UUID
	switch *ver {
	case 4:
		u = uuid.NewV4()
	case 7:
		u = uuid.NewV7()
	default:
		log.Fatalf("unsupported version %d; use 4 or 7", *ver)
	}
	fmt.Println(u)
}

// runParse parses a UUID string and prints structured details.
func runParse(args []string) {
	fs := flag.NewFlagSet("parse", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: uuid parse <uuid-string>")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Usage: uuid parse <uuid-string>")
		os.Exit(1)
	}

	u, err := uuid.Parse(fs.Arg(0))
	if err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Printf("uuid:    %s\n", u)
	fmt.Printf("version: %d\n", u.Version())
	fmt.Printf("variant: %s\n", u.Variant())
	fmt.Printf("nil:     %v\n", u.IsNil())
	fmt.Printf("max:     %v\n", u.IsMax())
}

// runVersion prints only the version number of a UUID string.
func runVersion(args []string) {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: uuid version <uuid-string>")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Usage: uuid version <uuid-string>")
		os.Exit(1)
	}

	u, err := uuid.Parse(fs.Arg(0))
	if err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Printf("%d\n", u.Version())
}
