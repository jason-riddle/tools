package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	pubclient "github.com/jason-riddle/tools/internal/pub/client"
)

const (
	defaultUser    = "jason-riddle"
	defaultTimeout = 10 * time.Second
	usage          = `pub - print public SSH keys for a GitHub user

Usage:
  pub [flags] [user]

Arguments:
  user    GitHub username to look up (default "jason-riddle")

Flags:
  -timeout duration
        HTTP timeout (default 10s)
  -h, -help, --help
        Show help

Examples:
  pub
  pub foobar-quz
  pub -timeout 5s octocat
`
)

var (
	errUsage      = errors.New("usage")
	githubBaseURL = "https://github.com"
)

type options struct {
	timeout time.Duration
	user    string
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("pub: ")

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

	body, err := pubclient.Fetch(opts.user, pubclient.Options{
		BaseURL: githubBaseURL,
		Timeout: opts.timeout,
	})
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(body)
	return err
}

func printUsageError(err error) {
	fmt.Fprintf(os.Stderr, "pub: %v\n\n%s", err, usage)
}

func parseOptions(args []string) (options, error) {
	var opts options

	fs := flag.NewFlagSet("pub", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { fmt.Fprint(os.Stdout, usage) }
	fs.DurationVar(&opts.timeout, "timeout", defaultTimeout, "HTTP timeout")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}

	switch fs.NArg() {
	case 0:
		opts.user = defaultUser
	case 1:
		opts.user = fs.Arg(0)
	default:
		return options{}, errors.New("accepts at most one username argument")
	}

	return opts, nil
}
