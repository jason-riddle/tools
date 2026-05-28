// Package client fetches GitHub public SSH keys.
package client

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// Options configures GitHub public key fetching.
type Options struct {
	BaseURL string
	Timeout time.Duration
}

// Fetch returns the public SSH keys for user from opts.BaseURL.
func Fetch(user string, opts Options) ([]byte, error) {
	httpClient := &http.Client{Timeout: opts.Timeout}
	url := opts.BaseURL + "/" + user + ".keys"

	resp, err := httpClient.Get(url)
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
