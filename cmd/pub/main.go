package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
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
	if wantsHelp(args) {
		fmt.Fprint(os.Stdout, usage)
		return nil
	}

	opts, err := parseOptions(args)
	if err != nil {
		printUsageError(err)
		return errUsage
	}

	client := &http.Client{Timeout: opts.timeout}
	body, err := fetchKeys(client, githubBaseURL, opts.user)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(body)
	return err
}

func printUsageError(err error) {
	fmt.Fprintf(os.Stderr, "pub: %v\n\n%s", err, usage)
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "-h", "--help", "-help":
			return true
		}
	}

	return false
}

func parseOptions(args []string) (options, error) {
	var opts options

	flagArgs, user, err := splitArgs(args)
	if err != nil {
		return options{}, err
	}

	fs := flag.NewFlagSet("pub", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.DurationVar(&opts.timeout, "timeout", defaultTimeout, "HTTP timeout")

	if err := fs.Parse(flagArgs); err != nil {
		return options{}, err
	}
	if fs.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	if user == "" {
		user = defaultUser
	}
	opts.user = user

	return opts, nil
}

func splitArgs(args []string) ([]string, string, error) {
	flagArgs := make([]string, 0, len(args))
	var user string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case hasFlagValue(arg, "timeout"):
			flagArgs = append(flagArgs, arg)
		case isValueFlag(arg, "timeout"):
			if i+1 >= len(args) {
				return nil, "", fmt.Errorf("flag needs a value: %s", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		case strings.HasPrefix(arg, "-"):
			flagArgs = append(flagArgs, arg)
		default:
			if user != "" {
				return nil, "", errors.New("accepts at most one username argument")
			}
			user = arg
		}
	}

	return flagArgs, user, nil
}

func isValueFlag(arg, name string) bool {
	return arg == "-"+name || arg == "--"+name
}

func hasFlagValue(arg, name string) bool {
	return strings.HasPrefix(arg, "-"+name+"=") || strings.HasPrefix(arg, "--"+name+"=")
}

func fetchKeys(client *http.Client, baseURL, user string) ([]byte, error) {
	url := strings.TrimRight(baseURL, "/") + "/" + user + ".keys"

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github returned %s for %q", resp.Status, user)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", url, err)
	}

	return body, nil
}
