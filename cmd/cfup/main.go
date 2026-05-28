package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/jason-riddle/tools/internal/cfup"
)

const (
	defaultTimeout = 10 * time.Second
	usage          = `cfup - check or proxy an HTTP service behind Cloudflare Access

Usage:
  cfup [flags] [url]

Modes:
  cfup https://example.com
  cfup --url https://example.com
  cfup --listen :8000 --url https://example.com

Flags:
  --url string
        Target upstream URL
  --listen string
        Listen address for proxy mode
  --timeout duration
        Per-request timeout (default 10s)
  -h, -help, --help
        Show help

Environment:
  CF_ACCESS_CLIENT_ID
  CF_ACCESS_CLIENT_SECRET

Examples:
  cfup https://example.com
  cfup --url https://example.com
  cfup --listen :8000 --url https://example.com
`
)

var errUsage = errors.New("usage")

type options struct {
	target  *url.URL
	listen  string
	timeout time.Duration
	token   cfup.AccessToken
}

type quietError struct {
	err error
}

func (e quietError) Error() string {
	return e.err.Error()
}

func (e quietError) Unwrap() error {
	return e.err
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("cfup: ")

	if err := run(os.Args[1:]); err != nil {
		var quiet quietError
		if !errors.Is(err, errUsage) && !errors.As(err, &quiet) {
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

	if opts.listen != "" {
		return runProxy(opts)
	}
	return runCheck(opts)
}

func runCheck(opts options) error {
	client := &http.Client{Timeout: opts.timeout}
	result, err := cfup.Check(context.Background(), client, opts.target, opts.token)
	if err != nil {
		return err
	}

	state := "healthy"
	if !result.Healthy {
		state = "unhealthy"
	}
	fmt.Printf("%s status=%d final_url=%s duration=%s\n", state, result.StatusCode, result.FinalURL, result.Duration.Round(time.Millisecond))
	if !result.Healthy {
		return quietError{err: fmt.Errorf("received unhealthy status %d", result.StatusCode)}
	}
	return nil
}

func runProxy(opts options) error {
	client := &http.Client{
		Timeout: opts.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	server := &http.Server{
		Addr:    opts.listen,
		Handler: cfup.NewProxyHandler(opts.target, client, opts.token),
	}

	log.Printf("listening on %s forwarding to %s", opts.listen, opts.target.Redacted())
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func printUsageError(err error) {
	fmt.Fprintf(os.Stderr, "cfup: %v\n\n%s", err, usage)
}

func parseOptions(args []string) (options, error) {
	var opts options

	fs := flag.NewFlagSet("cfup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { fmt.Fprint(os.Stdout, usage) }
	fs.StringVar(&opts.listen, "listen", "", "listen address for proxy mode")
	fs.DurationVar(&opts.timeout, "timeout", defaultTimeout, "per-request HTTP timeout")

	var rawURL string
	fs.StringVar(&rawURL, "url", "", "target upstream URL")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}

	opts.token = cfup.AccessToken{
		ClientID:     os.Getenv("CF_ACCESS_CLIENT_ID"),
		ClientSecret: os.Getenv("CF_ACCESS_CLIENT_SECRET"),
	}
	if (opts.token.ClientID == "") != (opts.token.ClientSecret == "") {
		return options{}, errors.New("CF_ACCESS_CLIENT_ID and CF_ACCESS_CLIENT_SECRET must both be set")
	}

	if opts.listen == "" {
		switch fs.NArg() {
		case 0:
		case 1:
			if rawURL != "" {
				return options{}, errors.New("use either --url or a positional URL, not both")
			}
			rawURL = fs.Arg(0)
		default:
			return options{}, errors.New("accepts at most one positional URL")
		}
	} else if fs.NArg() != 0 {
		return options{}, errors.New("proxy mode does not accept positional arguments; use --url")
	}

	if rawURL == "" {
		return options{}, errors.New("missing required --url or positional URL")
	}

	target, err := url.Parse(rawURL)
	if err != nil {
		return options{}, fmt.Errorf("parse URL: %w", err)
	}
	if target.Scheme == "" || target.Host == "" {
		return options{}, errors.New("URL must include scheme and host")
	}
	opts.target = target

	return opts, nil
}
