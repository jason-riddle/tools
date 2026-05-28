package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jason-riddle/tools/internal/nas"
)

const (
	defaultTimeout  = 10 * time.Second
	defaultInsecure = true
	usage           = `nas - interact with a Synology NAS via the DSM Web API

Usage:
  nas <command> [flags]

Commands:
  status    Print a summary of NAS health and resource usage
  reboot    Reboot the NAS

Run 'nas <command> -h' for command-specific help.

Environment:
  NAS_HOST      Hostname or IP with optional port (default port 5001)
  NAS_USER      DSM account username
  NAS_PASSWORD  DSM account password

Global help:
  -h, -help, --help
        Show help

Examples:
  nas status
  nas status -timeout 5s
  nas reboot
  nas reboot -confirm
`
)

var errUsage = errors.New("usage")

type options struct {
	timeout  time.Duration
	insecure bool
	host     string
	user     string
	password string
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("nas: ")

	if err := run(os.Args[1:], os.Stdin); err != nil {
		if !errors.Is(err, errUsage) {
			log.Print(err)
		}
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader) error {
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, usage)
		return errUsage
	}

	switch args[0] {
	case "status":
		return runStatus(args[1:])
	case "reboot":
		return runReboot(args[1:], stdin)
	case "-h", "-help", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
		return nil
	default:
		fmt.Fprintf(os.Stderr, "nas: unknown command %q\n\n", args[0])
		fmt.Fprint(os.Stderr, usage)
		return errUsage
	}
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { printStatusUsage(os.Stdout) }
	opts, err := parseCommonOptions(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		fmt.Fprintf(os.Stderr, "nas status: %v\n\n", err)
		printStatusUsage(os.Stderr)
		return errUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "nas status: unexpected arguments: %s\n\n", strings.Join(fs.Args(), " "))
		printStatusUsage(os.Stderr)
		return errUsage
	}

	client, err := nas.NewClient(nas.Options{
		Host:     opts.host,
		User:     opts.user,
		Password: opts.password,
		Timeout:  opts.timeout,
		Insecure: opts.insecure,
	})
	if err != nil {
		return err
	}
	if err := client.Login(opts.user, opts.password); err != nil {
		return mapAPIError(err)
	}
	defer func() { _ = client.Logout() }()

	status, err := client.Status()
	if err != nil {
		return mapAPIError(err)
	}

	usedKB := status.MemoryTotalKB - status.MemoryAvailKB
	fmt.Printf("Model:   %s\n", status.Model)
	fmt.Printf("DSM:     %s\n", status.DSMVersion)
	fmt.Printf("Uptime:  %s\n", formatUptime(status.UptimeSeconds))
	fmt.Printf("CPU:     user=%d%% sys=%d%%\n", status.CPUUser, status.CPUSystem)
	fmt.Printf("Memory:  used=%d MB / total=%d MB\n", kbToMB(usedKB), kbToMB(status.MemoryTotalKB))
	return nil
}

func runReboot(args []string, stdin io.Reader) error {
	fs := flag.NewFlagSet("reboot", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { printRebootUsage(os.Stdout) }
	confirm := fs.Bool("confirm", false, "skip confirmation prompt")
	opts, err := parseCommonOptions(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		fmt.Fprintf(os.Stderr, "nas reboot: %v\n\n", err)
		printRebootUsage(os.Stderr)
		return errUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "nas reboot: unexpected arguments: %s\n\n", strings.Join(fs.Args(), " "))
		printRebootUsage(os.Stderr)
		return errUsage
	}

	if !*confirm {
		ok, err := confirmReboot(stdin, opts.host)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("reboot cancelled")
		}
	}

	client, err := nas.NewClient(nas.Options{
		Host:     opts.host,
		User:     opts.user,
		Password: opts.password,
		Timeout:  opts.timeout,
		Insecure: opts.insecure,
	})
	if err != nil {
		return err
	}
	if err := client.Login(opts.user, opts.password); err != nil {
		return mapAPIError(err)
	}
	defer func() { _ = client.Logout() }()

	if err := client.Reboot(); err != nil {
		return mapAPIError(err)
	}

	fmt.Println("reboot initiated")
	return nil
}

func parseCommonOptions(fs *flag.FlagSet, args []string) (options, error) {
	var opts options
	fs.DurationVar(&opts.timeout, "timeout", defaultTimeout, "per-request HTTP timeout")
	fs.BoolVar(&opts.insecure, "insecure", defaultInsecure, "skip TLS certificate verification")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}

	opts.host = os.Getenv("NAS_HOST")
	opts.user = os.Getenv("NAS_USER")
	opts.password = os.Getenv("NAS_PASSWORD")

	missing := missingEnv(opts)
	if len(missing) > 0 {
		return options{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return opts, nil
}

func missingEnv(opts options) []string {
	var missing []string
	if opts.host == "" {
		missing = append(missing, "NAS_HOST")
	}
	if opts.user == "" {
		missing = append(missing, "NAS_USER")
	}
	if opts.password == "" {
		missing = append(missing, "NAS_PASSWORD")
	}
	return missing
}

func confirmReboot(stdin io.Reader, host string) (bool, error) {
	fmt.Printf("Reboot %s? [y/N] ", host)
	reader := bufio.NewReader(stdin)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	return strings.TrimSpace(line) == "y", nil
}

func formatUptime(seconds int64) string {
	d := time.Duration(seconds) * time.Second
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func kbToMB(kb int64) int64 {
	return kb / 1024
}

func mapAPIError(err error) error {
	var apiErr *nas.APIError
	if !errors.As(err, &apiErr) {
		return err
	}

	switch apiErr.Code {
	case 105:
		return fmt.Errorf("administrator privileges required")
	case 106:
		return fmt.Errorf("DSM session timed out")
	case 107:
		return fmt.Errorf("DSM session interrupted by duplicate login")
	case 119:
		return fmt.Errorf("DSM session is invalid")
	case 400:
		return fmt.Errorf("incorrect username or password")
	case 403:
		return fmt.Errorf("2FA code required and not supported")
	default:
		return err
	}
}

func printStatusUsage(w io.Writer) {
	fmt.Fprint(w, `nas status - print NAS health and resource usage

Usage:
  nas status [flags]

Flags:
  -timeout duration
        Per-request HTTP timeout (default 10s)
  -insecure
        Skip TLS certificate verification (default true)

Environment:
  NAS_HOST      Hostname or IP with optional port (default port 5001)
  NAS_USER      DSM account username
  NAS_PASSWORD  DSM account password

Examples:
  nas status
  nas status -timeout 5s
`)
}

func printRebootUsage(w io.Writer) {
	fmt.Fprint(w, `nas reboot - reboot the NAS

Usage:
  nas reboot [flags]

Flags:
  -confirm
        Skip confirmation prompt
  -timeout duration
        Per-request HTTP timeout (default 10s)
  -insecure
        Skip TLS certificate verification (default true)

Environment:
  NAS_HOST      Hostname or IP with optional port (default port 5001)
  NAS_USER      DSM account username
  NAS_PASSWORD  DSM account password

Examples:
  nas reboot
  nas reboot -confirm
`)
}
