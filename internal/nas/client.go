// Package nas provides a small Synology DSM Web API client.
package nas

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPSPort = "5001"
	authVersion      = 7
)

// Options configures DSM API access.
type Options struct {
	Host     string
	User     string
	Password string
	Timeout  time.Duration
	Insecure bool
	BaseURL  string
}

// Status combines the fields needed by the CLI summary output.
type Status struct {
	Model         string
	DSMVersion    string
	UptimeSeconds int64
	CPUUser       int
	CPUSystem     int
	MemoryTotalKB int64
	MemoryAvailKB int64
}

// Client talks to a Synology DSM Web API endpoint.
type Client struct {
	baseURL    string
	httpClient *http.Client
	sid        string
	token      string
}

type apiResponse[T any] struct {
	Data    T        `json:"data"`
	Error   apiError `json:"error"`
	Success bool     `json:"success"`
}

type apiError struct {
	Code int `json:"code"`
}

type loginResponse struct {
	SID       string `json:"sid"`
	SynoToken string `json:"synotoken"`
}

type dsmInfoResponse struct {
	Model         string `json:"model"`
	VersionString string `json:"version_string"`
	Uptime        int64  `json:"uptime"`
}

type utilizationResponse struct {
	CPU struct {
		UserLoad   int `json:"user_load"`
		SystemLoad int `json:"system_load"`
	} `json:"cpu"`
	Memory struct {
		TotalReal int64 `json:"total_real"`
		AvailReal int64 `json:"avail_real"`
	} `json:"memory"`
}

// APIError reports a DSM API error code with the API name that returned it.
type APIError struct {
	API  string
	Code int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s returned error code %d", e.API, e.Code)
}

// NewClient constructs a DSM client with sensible defaults for Synology boxes.
func NewClient(opts Options) (*Client, error) {
	baseURL, err := normalizeBaseURL(opts)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: opts.Insecure} //nolint:gosec // Synology devices commonly use self-signed certificates.

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout:   opts.Timeout,
			Transport: transport,
		},
	}, nil
}

// Login authenticates and stores the DSM session identifiers for later requests.
func (c *Client) Login(user, password string) error {
	var data loginResponse
	if err := c.doRequest(http.MethodGet, "SYNO.API.Auth", authVersion, "login", map[string]string{
		"account":           user,
		"passwd":            password,
		"format":            "sid",
		"enable_syno_token": "yes",
		"session":           "Core",
	}, &data, false); err != nil {
		return err
	}

	if data.SID == "" {
		return fmt.Errorf("SYNO.API.Auth login response missing sid")
	}
	if data.SynoToken == "" {
		return fmt.Errorf("SYNO.API.Auth login response missing synotoken")
	}

	c.sid = data.SID
	c.token = data.SynoToken
	return nil
}

// Logout releases the DSM session. It is safe to call on an unauthenticated client.
func (c *Client) Logout() error {
	if c.sid == "" {
		return nil
	}

	err := c.doRequest(http.MethodGet, "SYNO.API.Auth", authVersion, "logout", map[string]string{
		"_sid": c.sid,
	}, nil, false)
	c.sid = ""
	c.token = ""
	return err
}

// Status fetches a summary of model, version, uptime, CPU, and memory usage.
func (c *Client) Status() (Status, error) {
	var info dsmInfoResponse
	if err := c.doRequest(http.MethodGet, "SYNO.DSM.Info", 2, "getinfo", nil, &info, true); err != nil {
		return Status{}, err
	}

	var util utilizationResponse
	if err := c.doRequest(http.MethodGet, "SYNO.Core.System.Utilization", 1, "get", nil, &util, true); err != nil {
		return Status{}, err
	}

	return Status{
		Model:         info.Model,
		DSMVersion:    info.VersionString,
		UptimeSeconds: info.Uptime,
		CPUUser:       util.CPU.UserLoad,
		CPUSystem:     util.CPU.SystemLoad,
		MemoryTotalKB: util.Memory.TotalReal,
		MemoryAvailKB: util.Memory.AvailReal,
	}, nil
}

// Reboot instructs DSM to reboot immediately.
func (c *Client) Reboot() error {
	return c.doRequest(http.MethodGet, "SYNO.Core.System", 1, "reboot", nil, nil, true)
}

func (c *Client) doRequest(method, api string, version int, action string, extra map[string]string, out any, includeSession bool) error {
	if includeSession && (c.sid == "" || c.token == "") {
		return fmt.Errorf("not logged in")
	}

	values := url.Values{}
	values.Set("api", api)
	values.Set("version", strconv.Itoa(version))
	values.Set("method", action)

	if includeSession {
		values.Set("_sid", c.sid)
		values.Set("SynoToken", c.token)
	}

	for key, value := range extra {
		values.Set(key, value)
	}

	requestURL := c.baseURL + "/webapi/entry.cgi?" + values.Encode()
	req, err := http.NewRequest(method, requestURL, nil)
	if err != nil {
		return fmt.Errorf("build %s request: %w", api, err)
	}
	if includeSession {
		req.Header.Set("X-SYNO-TOKEN", c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %s: %w", api, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %s", api, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read %s response: %w", api, err)
	}

	if out == nil {
		var parsed apiResponse[json.RawMessage]
		if err := json.Unmarshal(body, &parsed); err != nil {
			return fmt.Errorf("decode %s response: %w", api, err)
		}
		if !parsed.Success {
			return &APIError{API: api, Code: parsed.Error.Code}
		}
		return nil
	}

	var parsed apiResponse[json.RawMessage]
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("decode %s response: %w", api, err)
	}
	if !parsed.Success {
		return &APIError{API: api, Code: parsed.Error.Code}
	}
	if len(parsed.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(parsed.Data, out); err != nil {
		return fmt.Errorf("decode %s data: %w", api, err)
	}
	return nil
}

func normalizeBaseURL(opts Options) (string, error) {
	if opts.BaseURL != "" {
		return strings.TrimRight(opts.BaseURL, "/"), nil
	}
	if opts.Host == "" {
		return "", fmt.Errorf("host is required")
	}

	host := opts.Host
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}

	parsed, err := url.Parse(host)
	if err != nil {
		return "", fmt.Errorf("parse host %q: %w", opts.Host, err)
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}
	if parsed.Host == "" {
		parsed.Host = parsed.Path
		parsed.Path = ""
	}

	if parsed.Port() == "" {
		parsed.Host = net.JoinHostPort(parsed.Hostname(), defaultHTTPSPort)
	}

	parsed.Path = path.Clean(parsed.Path)
	if parsed.Path == "." {
		parsed.Path = ""
	}

	return strings.TrimRight(parsed.String(), "/"), nil
}
