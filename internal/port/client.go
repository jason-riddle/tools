// Package port talks to the Portainer HTTP API.
package port

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Options configures a Portainer client.
type Options struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// Client is a minimal Portainer API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Status describes the Portainer server instance.
type Status struct {
	Version    string `json:"Version"`
	InstanceID string `json:"InstanceID"`
}

// Endpoint describes a Portainer-managed environment.
type Endpoint struct {
	ID        int                `json:"Id"`
	Name      string             `json:"Name"`
	Type      int                `json:"Type"`
	URL       string             `json:"URL"`
	PublicURL string             `json:"PublicURL"`
	Status    int                `json:"Status"`
	Snapshots []EndpointSnapshot `json:"Snapshots"`
}

// EndpointSnapshot contains the latest environment snapshot Portainer exposes.
type EndpointSnapshot struct {
	Time                    int64  `json:"Time"`
	DockerVersion           string `json:"DockerVersion"`
	TotalCPU                int    `json:"TotalCPU"`
	TotalMemory             int64  `json:"TotalMemory"`
	ContainerCount          int    `json:"ContainerCount"`
	RunningContainerCount   int    `json:"RunningContainerCount"`
	StoppedContainerCount   int    `json:"StoppedContainerCount"`
	HealthyContainerCount   int    `json:"HealthyContainerCount"`
	UnhealthyContainerCount int    `json:"UnhealthyContainerCount"`
	VolumeCount             int    `json:"VolumeCount"`
	ImageCount              int    `json:"ImageCount"`
	ServiceCount            int    `json:"ServiceCount"`
	StackCount              int    `json:"StackCount"`
}

type apiError struct {
	Message string `json:"message"`
	Details string `json:"details"`
}

// NewClient returns a client configured for a Portainer server.
func NewClient(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("missing Portainer base URL")
	}
	if opts.APIKey == "" {
		return nil, fmt.Errorf("missing Portainer API key")
	}

	parsed, err := url.Parse(opts.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse Portainer base URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("Portainer base URL must include scheme and host")
	}

	baseURL := strings.TrimRight(parsed.String(), "/")
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &Client{
		baseURL:    baseURL,
		apiKey:     opts.APIKey,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

// Status returns server metadata from /api/status.
func (c *Client) Status(ctx context.Context) (Status, error) {
	var status Status
	if err := c.getJSON(ctx, "/api/status", &status); err != nil {
		return Status{}, err
	}
	return status, nil
}

// Endpoints returns the configured Portainer endpoints.
func (c *Client) Endpoints(ctx context.Context) ([]Endpoint, error) {
	var endpoints []Endpoint
	if err := c.getJSON(ctx, "/api/endpoints", &endpoints); err != nil {
		return nil, err
	}
	return endpoints, nil
}

func (c *Client) getJSON(ctx context.Context, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build %s request: %w", path, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return fmt.Errorf("GET %s: %s", path, resp.Status)
		}
		var apiErr apiError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
			if apiErr.Details != "" {
				return fmt.Errorf("GET %s: %s: %s", path, apiErr.Message, apiErr.Details)
			}
			return fmt.Errorf("GET %s: %s", path, apiErr.Message)
		}
		return fmt.Errorf("GET %s: %s", path, resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode %s response: %w", path, err)
	}
	return nil
}
