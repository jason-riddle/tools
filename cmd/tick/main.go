package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

const usage = `tick - print the current time

Usage:
  tick [flags] [offset]

Arguments:
  offset  Optional duration offset such as +24h, -90m, or +30s.
          The offset may appear before or after flags.

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
  tick -nano -1h
  tick -format '2006-01-02 15:04:05 MST' +30m
  TZ=America/New_York tick
`

var errUsage = errors.New("usage")

type options struct {
	nano      bool
	epoch     bool
	json      bool
	format    string
	hasOffset bool
	offset    time.Duration
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
	if wantsHelp(args) {
		fmt.Fprint(os.Stdout, usage)
		return nil
	}

	opts, err := parseOptions(args)
	if err != nil {
		printUsageError(err)
		return errUsage
	}

	loc, err := locationFromEnv(os.LookupEnv("TZ"))
	if err != nil {
		return err
	}

	now := time.Now().In(loc)
	if opts.hasOffset {
		now = now.Add(opts.offset)
	}

	output, err := formatTime(now, opts)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, output)

	return nil
}

func printUsageError(err error) {
	fmt.Fprintf(os.Stderr, "tick: %v\n\n%s", err, usage)
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

	flagArgs, offset, err := splitArgs(args)
	if err != nil {
		return options{}, err
	}

	fs := flag.NewFlagSet("tick", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&opts.nano, "nano", false, "print in RFC3339Nano format")
	fs.BoolVar(&opts.epoch, "epoch", false, "print Unix epoch seconds")
	fs.BoolVar(&opts.json, "json", false, "print common time package layouts as JSON")
	fs.StringVar(&opts.format, "format", "", "print using a Go time layout string")

	if err := fs.Parse(flagArgs); err != nil {
		return options{}, err
	}
	if fs.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	modeCount := 0
	if opts.nano {
		modeCount++
	}
	if opts.epoch {
		modeCount++
	}
	if opts.json {
		modeCount++
	}
	if opts.format != "" {
		modeCount++
	}
	if modeCount > 1 {
		return options{}, errors.New("-nano, -epoch, -format, and -json are mutually exclusive")
	}

	if offset != nil {
		opts.hasOffset = true
		opts.offset = *offset
	}

	return opts, nil
}

func splitArgs(args []string) ([]string, *time.Duration, error) {
	flagArgs := make([]string, 0, len(args))
	var offset *time.Duration

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case isBoolFlag(arg), isHelpFlag(arg), hasFlagValue(arg, "format"):
			flagArgs = append(flagArgs, arg)
		case isValueFlag(arg, "format"):
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag needs a value: %s", arg)
			}
			flagArgs = append(flagArgs, arg, args[i+1])
			i++
		default:
			dur, err := time.ParseDuration(arg)
			if err == nil {
				if offset != nil {
					return nil, nil, errors.New("only one offset argument is allowed")
				}
				offset = &dur
				continue
			}

			if strings.HasPrefix(arg, "-") {
				flagArgs = append(flagArgs, arg)
				continue
			}

			return nil, nil, fmt.Errorf("unexpected argument %q", arg)
		}
	}

	return flagArgs, offset, nil
}

func isBoolFlag(arg string) bool {
	switch arg {
	case "-nano", "--nano", "-epoch", "--epoch", "-json", "--json":
		return true
	}

	return false
}

func isHelpFlag(arg string) bool {
	switch arg {
	case "-h", "--help", "-help":
		return true
	}

	return false
}

func isValueFlag(arg, name string) bool {
	return arg == "-"+name || arg == "--"+name
}

func hasFlagValue(arg, name string) bool {
	return strings.HasPrefix(arg, "-"+name+"=") || strings.HasPrefix(arg, "--"+name+"=")
}

func locationFromEnv(tz string, ok bool) (*time.Location, error) {
	if !ok {
		return time.UTC, nil
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("load TZ location %q: %w", tz, err)
	}

	return loc, nil
}

func formatTime(t time.Time, opts options) (string, error) {
	switch {
	case opts.nano:
		return t.Format(time.RFC3339Nano), nil
	case opts.epoch:
		return fmt.Sprintf("%d", t.Unix()), nil
	case opts.format != "":
		return t.Format(opts.format), nil
	case opts.json:
		return jsonTime(t)
	default:
		return t.Format(time.RFC3339), nil
	}
}

func jsonTime(t time.Time) (string, error) {
	data := map[string]any{
		"ANSIC":       t.Format(time.ANSIC),
		"DateOnly":    t.Format(time.DateOnly),
		"DateTime":    t.Format(time.DateTime),
		"Kitchen":     t.Format(time.Kitchen),
		"RFC822":      t.Format(time.RFC822),
		"RFC822Z":     t.Format(time.RFC822Z),
		"RFC850":      t.Format(time.RFC850),
		"RFC1123":     t.Format(time.RFC1123),
		"RFC1123Z":    t.Format(time.RFC1123Z),
		"RFC3339":     t.Format(time.RFC3339),
		"RFC3339Nano": t.Format(time.RFC3339Nano),
		"RubyDate":    t.Format(time.RubyDate),
		"Stamp":       t.Format(time.Stamp),
		"StampMicro":  t.Format(time.StampMicro),
		"StampMilli":  t.Format(time.StampMilli),
		"StampNano":   t.Format(time.StampNano),
		"TimeOnly":    t.Format(time.TimeOnly),
		"UnixDate":    t.Format(time.UnixDate),
		"epoch":       t.Unix(),
	}

	b, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal json output: %w", err)
	}

	return string(b), nil
}
