package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jason-riddle/tools/internal/grab"
)

const (
	defaultTimeout = 30 * time.Second
	usage          = `grab - download a file into the current directory

Usage:
  grab [flags] <url>

Arguments:
  url     URL to download

Flags:
  -o string
        Output filename
  -f
        Allow overwriting an existing file
  -timeout duration
        HTTP timeout (default 30s)
  -h, -help, --help
        Show help

Examples:
  grab https://example.com/file.tar.gz
  grab -o archive.tar.gz https://example.com/file.tar.gz
`
)

var errUsage = errors.New("usage")

type options struct {
	output  string
	force   bool
	timeout time.Duration
	url     string
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("grab: ")

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

	client := &http.Client{Timeout: opts.timeout}
	savedAs, err := grab.Download(context.Background(), client, opts.url, grab.Options{
		Output: opts.output,
		Force:  opts.force,
	})
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(os.Stdout, savedAs)
	return err
}

func printUsageError(err error) {
	fmt.Fprintf(os.Stderr, "grab: %v\n\n%s", err, usage)
}

func parseOptions(args []string) (options, error) {
	var opts options

	fs := flag.NewFlagSet("grab", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { fmt.Fprint(os.Stdout, usage) }
	fs.StringVar(&opts.output, "o", "", "output filename")
	fs.BoolVar(&opts.force, "f", false, "allow overwriting an existing file")
	fs.DurationVar(&opts.timeout, "timeout", defaultTimeout, "HTTP timeout")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}

	if fs.NArg() != 1 {
		return options{}, errors.New("accepts exactly one URL argument")
	}
	if opts.timeout <= 0 {
		return options{}, errors.New("timeout must be greater than zero")
	}

	opts.url = fs.Arg(0)
	return opts, nil
}
