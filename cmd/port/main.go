package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	portclient "github.com/jason-riddle/tools/internal/port"
)

const (
	defaultTimeout = 10 * time.Second
	usage          = `port - interact with Portainer

Usage:
  port <command> [flags]

Commands:
  status    Print Portainer server and endpoint status

Run 'port <command> -h' for command-specific help.

Environment:
  PORTAINER_URL      Portainer base URL, for example http://nas:9000
  PORTAINER_API_KEY  Portainer API key sent as X-API-Key

Global help:
  -h, -help, --help
        Show help

Examples:
  port status
  port status --url http://nas:9000
  port status --json
`
)

var errUsage = errors.New("usage")

type options struct {
	url     string
	apiKey  string
	timeout time.Duration
	json    bool
}

type statusOutput struct {
	URL           string           `json:"url"`
	Version       string           `json:"version"`
	InstanceID    string           `json:"instance_id"`
	Authenticated bool             `json:"authenticated"`
	EndpointCount int              `json:"endpoint_count"`
	Endpoints     []endpointOutput `json:"endpoints"`
}

type endpointOutput struct {
	ID        int             `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	URL       string          `json:"url,omitempty"`
	PublicURL string          `json:"public_url,omitempty"`
	Status    string          `json:"status"`
	Snapshot  *snapshotOutput `json:"snapshot,omitempty"`
}

type snapshotOutput struct {
	Time                string `json:"time"`
	DockerVersion       string `json:"docker_version,omitempty"`
	TotalCPU            int    `json:"total_cpu,omitempty"`
	TotalMemoryBytes    int64  `json:"total_memory_bytes,omitempty"`
	ContainerCount      int    `json:"container_count"`
	RunningContainers   int    `json:"running_containers"`
	StoppedContainers   int    `json:"stopped_containers"`
	HealthyContainers   int    `json:"healthy_containers"`
	UnhealthyContainers int    `json:"unhealthy_containers"`
	VolumeCount         int    `json:"volume_count"`
	ImageCount          int    `json:"image_count"`
	ServiceCount        int    `json:"service_count"`
	StackCount          int    `json:"stack_count"`
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("port: ")

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
	case "status":
		return runStatus(args[1:])
	case "-h", "-help", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
		return nil
	default:
		fmt.Fprintf(os.Stderr, "port: unknown command %q\n\n", args[0])
		fmt.Fprint(os.Stderr, usage)
		return errUsage
	}
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { printStatusUsage(os.Stdout) }
	opts, err := parseStatusOptions(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		fmt.Fprintf(os.Stderr, "port status: %v\n\n", err)
		printStatusUsage(os.Stderr)
		return errUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "port status: unexpected arguments: %s\n\n", strings.Join(fs.Args(), " "))
		printStatusUsage(os.Stderr)
		return errUsage
	}

	client, err := portclient.NewClient(portclient.Options{
		BaseURL: opts.url,
		APIKey:  opts.apiKey,
		Timeout: opts.timeout,
	})
	if err != nil {
		return err
	}

	ctx := context.Background()
	status, err := client.Status(ctx)
	if err != nil {
		return err
	}
	endpoints, err := client.Endpoints(ctx)
	if err != nil {
		return err
	}

	out := buildStatusOutput(opts.url, status, endpoints)
	if opts.json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	printHumanStatus(os.Stdout, out)
	return nil
}

func parseStatusOptions(fs *flag.FlagSet, args []string) (options, error) {
	var opts options
	fs.StringVar(&opts.url, "url", "", "Portainer base URL")
	fs.StringVar(&opts.apiKey, "api-key", "", "Portainer API key")
	fs.DurationVar(&opts.timeout, "timeout", defaultTimeout, "per-request HTTP timeout")
	fs.BoolVar(&opts.json, "json", false, "emit JSON output")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}

	if opts.url == "" {
		opts.url = os.Getenv("PORTAINER_URL")
	}
	if opts.apiKey == "" {
		opts.apiKey = os.Getenv("PORTAINER_API_KEY")
	}
	if opts.url == "" {
		return options{}, fmt.Errorf("missing Portainer URL; set --url or PORTAINER_URL")
	}
	if opts.apiKey == "" {
		return options{}, fmt.Errorf("missing Portainer API key; set --api-key or PORTAINER_API_KEY")
	}

	return opts, nil
}

func buildStatusOutput(baseURL string, status portclient.Status, endpoints []portclient.Endpoint) statusOutput {
	out := statusOutput{
		URL:           baseURL,
		Version:       status.Version,
		InstanceID:    status.InstanceID,
		Authenticated: true,
		EndpointCount: len(endpoints),
		Endpoints:     make([]endpointOutput, 0, len(endpoints)),
	}

	for _, endpoint := range endpoints {
		item := endpointOutput{
			ID:        endpoint.ID,
			Name:      endpoint.Name,
			Type:      endpointType(endpoint.Type),
			URL:       endpoint.URL,
			PublicURL: endpoint.PublicURL,
			Status:    endpointStatus(endpoint.Status),
		}
		if snapshot, ok := latestSnapshot(endpoint.Snapshots); ok {
			item.Snapshot = &snapshotOutput{
				Time:                time.Unix(snapshot.Time, 0).UTC().Format(time.RFC3339),
				DockerVersion:       snapshot.DockerVersion,
				TotalCPU:            snapshot.TotalCPU,
				TotalMemoryBytes:    snapshot.TotalMemory,
				ContainerCount:      snapshot.ContainerCount,
				RunningContainers:   snapshot.RunningContainerCount,
				StoppedContainers:   snapshot.StoppedContainerCount,
				HealthyContainers:   snapshot.HealthyContainerCount,
				UnhealthyContainers: snapshot.UnhealthyContainerCount,
				VolumeCount:         snapshot.VolumeCount,
				ImageCount:          snapshot.ImageCount,
				ServiceCount:        snapshot.ServiceCount,
				StackCount:          snapshot.StackCount,
			}
		}
		out.Endpoints = append(out.Endpoints, item)
	}

	return out
}

func latestSnapshot(snapshots []portclient.EndpointSnapshot) (portclient.EndpointSnapshot, bool) {
	if len(snapshots) == 0 {
		return portclient.EndpointSnapshot{}, false
	}
	latest := snapshots[0]
	for _, snapshot := range snapshots[1:] {
		if snapshot.Time > latest.Time {
			latest = snapshot
		}
	}
	return latest, true
}

func endpointType(value int) string {
	switch value {
	case 1:
		return "docker"
	case 2:
		return "agent"
	case 3:
		return "azure"
	case 4:
		return "edge-agent"
	case 5:
		return "kubernetes"
	default:
		return fmt.Sprintf("unknown(%d)", value)
	}
}

func endpointStatus(value int) string {
	switch value {
	case 1:
		return "up"
	case 2:
		return "down"
	default:
		return fmt.Sprintf("unknown(%d)", value)
	}
}

func printHumanStatus(w io.Writer, out statusOutput) {
	fmt.Fprintf(w, "Portainer: %s\n", out.URL)
	fmt.Fprintf(w, "Version:   %s\n", out.Version)
	fmt.Fprintf(w, "Auth:      ok\n")
	fmt.Fprintf(w, "Endpoints: %d\n", out.EndpointCount)

	for _, endpoint := range out.Endpoints {
		fmt.Fprintf(w, "\n%s (id=%d, type=%s, status=%s)\n", endpoint.Name, endpoint.ID, endpoint.Type, endpoint.Status)
		if endpoint.URL != "" {
			fmt.Fprintf(w, "  URL:          %s\n", endpoint.URL)
		}
		if endpoint.PublicURL != "" {
			fmt.Fprintf(w, "  Public URL:   %s\n", endpoint.PublicURL)
		}
		if endpoint.Snapshot != nil {
			fmt.Fprintf(w, "  Snapshot:     %s\n", endpoint.Snapshot.Time)
			if endpoint.Snapshot.DockerVersion != "" {
				fmt.Fprintf(w, "  Docker:       %s\n", endpoint.Snapshot.DockerVersion)
			}
			fmt.Fprintf(w, "  Containers:   total=%d running=%d stopped=%d\n", endpoint.Snapshot.ContainerCount, endpoint.Snapshot.RunningContainers, endpoint.Snapshot.StoppedContainers)
			fmt.Fprintf(w, "  Health:       healthy=%d unhealthy=%d\n", endpoint.Snapshot.HealthyContainers, endpoint.Snapshot.UnhealthyContainers)
			fmt.Fprintf(w, "  Images:       %d\n", endpoint.Snapshot.ImageCount)
			fmt.Fprintf(w, "  Volumes:      %d\n", endpoint.Snapshot.VolumeCount)
			fmt.Fprintf(w, "  Stacks:       %d\n", endpoint.Snapshot.StackCount)
			fmt.Fprintf(w, "  Services:     %d\n", endpoint.Snapshot.ServiceCount)
		}
	}
}

func printStatusUsage(w io.Writer) {
	fmt.Fprint(w, `port status - print Portainer server and endpoint status

Usage:
  port status [flags]

Flags:
  --url string
        Portainer base URL
  --api-key string
        Portainer API key
  --timeout duration
        Per-request HTTP timeout (default 10s)
  --json
        Emit JSON output

Environment:
  PORTAINER_URL      Portainer base URL, for example http://nas:9000
  PORTAINER_API_KEY  Portainer API key sent as X-API-Key

Examples:
  port status
  port status --url http://nas:9000
  port status --json
`)
}
