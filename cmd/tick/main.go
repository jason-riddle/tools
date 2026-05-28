package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	tickformat "github.com/jason-riddle/tools/internal/tick/format"
)

const usage = `tick - print the current time

Usage:
  tick [flags] [offset]

Arguments:
  offset  Optional duration offset such as +24h or +30s.

Flags:
  -nano
        Print in RFC3339Nano format
  -epoch
        Print Unix epoch seconds
  -format string
        Print using a Go time layout string
  -json
        Print common time package layouts as JSON
  -h, -help, --help
        Show help

Notes:
  -nano, -epoch, -format, and -json are mutually exclusive.

Examples:
  tick
  tick +24h
  tick -nano
  tick -format '2006-01-02 15:04:05 MST' +30m
  TZ=America/New_York tick
`

var errUsage = errors.New("usage")

type options struct {
	mode   tickformat.Mode
	layout string
	offset time.Duration
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("tick: ")

	if err := run(os.Args[1:]); err != nil {
		if !errors.Is(err, errUsage) {
			log.Print(err)
		}
		os.Exit(1)
	}
}

func run(args []string) error {
	opts, err := parseOptions(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		printUsageError(err)
		return errUsage
	}

	loc, err := tickformat.Location(os.Getenv("TZ"))
	if err != nil {
		return err
	}

	now := time.Now().In(loc)
	fmt.Fprintln(os.Stdout, tickformat.Format(now, tickformat.Options{
		Mode:   opts.mode,
		Layout: opts.layout,
		Offset: opts.offset,
	}))

	return nil
}

func printUsageError(err error) {
	fmt.Fprintf(os.Stderr, "tick: %v\n\n%s", err, usage)
}

func parseOptions(args []string) (options, error) {
	var opts options
	var nano bool
	var epoch bool
	var jsonOutput bool

	fs := flag.NewFlagSet("tick", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { fmt.Fprint(os.Stdout, usage) }
	fs.BoolVar(&nano, "nano", false, "print in RFC3339Nano format")
	fs.BoolVar(&epoch, "epoch", false, "print Unix epoch seconds")
	fs.BoolVar(&jsonOutput, "json", false, "print common time package layouts as JSON")
	fs.StringVar(&opts.layout, "format", "", "print using a Go time layout string")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}

	modeCount := 0
	if nano {
		modeCount++
	}
	if epoch {
		modeCount++
	}
	if jsonOutput {
		modeCount++
	}
	if opts.layout != "" {
		modeCount++
	}
	if modeCount > 1 {
		return options{}, errors.New("-nano, -epoch, -format, and -json are mutually exclusive")
	}

	switch {
	case nano:
		opts.mode = tickformat.ModeRFC3339Nano
	case epoch:
		opts.mode = tickformat.ModeEpoch
	case jsonOutput:
		opts.mode = tickformat.ModeJSON
	case opts.layout != "":
		opts.mode = tickformat.ModeLayout
	default:
		opts.mode = tickformat.ModeRFC3339
	}

	switch fs.NArg() {
	case 0:
	case 1:
		dur, err := time.ParseDuration(fs.Arg(0))
		if err != nil {
			return options{}, fmt.Errorf("invalid offset %q: %w", fs.Arg(0), err)
		}
		opts.offset = dur
	default:
		return options{}, fmt.Errorf("unexpected arguments: %s", fs.Args())
	}

	return opts, nil
}
