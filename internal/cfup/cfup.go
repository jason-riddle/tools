// Package cfup checks and proxies HTTP services behind Cloudflare Access.
package cfup

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AccessToken carries the optional Cloudflare Access service token.
type AccessToken struct {
	ClientID     string
	ClientSecret string
}

// CheckResult reports the final response observed for a health check.
type CheckResult struct {
	StatusCode int
	FinalURL   string
	Duration   time.Duration
	Healthy    bool
	ReachedApp bool
}

// Check performs a GET against target, follows redirects, and reports the final response.
func Check(ctx context.Context, client *http.Client, target *url.URL, token AccessToken) (CheckResult, error) {
	start := time.Now()
	req, err := NewCheckRequest(ctx, target, token)
	if err != nil {
		return CheckResult{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{}, fmt.Errorf("request %s: %w", target.Redacted(), err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return CheckResult{}, fmt.Errorf("read %s response: %w", target.Redacted(), err)
	}

	finalURL := req.URL.String()
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	reachedApp := reachedApplication(target, resp)

	return CheckResult{
		StatusCode: resp.StatusCode,
		FinalURL:   finalURL,
		Duration:   time.Since(start),
		Healthy:    isHealthy(resp.StatusCode) && reachedApp,
		ReachedApp: reachedApp,
	}, nil
}

// NewCheckRequest builds a GET request to target and injects the optional Access token.
func NewCheckRequest(ctx context.Context, target *url.URL, token AccessToken) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	applyAccessHeaders(req.Header, token)
	return req, nil
}

// NewProxyHandler returns an HTTP handler that forwards requests to base.
func NewProxyHandler(base *url.URL, client *http.Client, token AccessToken) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamURL := ResolveProxyURL(base, r.URL)
		upstreamReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL.String(), r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("build upstream request: %v", err), http.StatusBadGateway)
			return
		}

		copyHeader(upstreamReq.Header, r.Header)
		removeHopByHopHeaders(upstreamReq.Header)
		applyAccessHeaders(upstreamReq.Header, token)
		upstreamReq.ContentLength = r.ContentLength

		resp, err := client.Do(upstreamReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("proxy request failed: %v", err), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		copyHeader(w.Header(), resp.Header)
		removeHopByHopHeaders(w.Header())
		w.WriteHeader(resp.StatusCode)
		if _, err := io.Copy(w, resp.Body); err != nil {
			return
		}
	})
}

// ResolveProxyURL combines the configured upstream base URL with an incoming request URL.
func ResolveProxyURL(base, incoming *url.URL) *url.URL {
	resolved := *base
	resolved.Path = joinURLPath(base.Path, incoming.Path)
	resolved.RawPath = ""
	resolved.RawQuery = joinQuery(base.RawQuery, incoming.RawQuery)
	resolved.Fragment = ""
	return &resolved
}

func applyAccessHeaders(header http.Header, token AccessToken) {
	if token.ClientID == "" || token.ClientSecret == "" {
		return
	}

	header.Set("CF-Access-Client-Id", token.ClientID)
	header.Set("CF-Access-Client-Secret", token.ClientSecret)
}

func isHealthy(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode < http.StatusBadRequest
}

func reachedApplication(target *url.URL, resp *http.Response) bool {
	if resp.Request == nil || resp.Request.URL == nil {
		return false
	}

	finalURL := resp.Request.URL
	if strings.EqualFold(resp.Header.Get("Cf-Access-Domain"), target.Hostname()) {
		return false
	}
	if strings.HasSuffix(strings.ToLower(finalURL.Hostname()), ".cloudflareaccess.com") {
		return false
	}
	if strings.HasPrefix(finalURL.Path, "/cdn-cgi/access/") {
		return false
	}
	return true
}

func joinURLPath(basePath, requestPath string) string {
	if requestPath == "" {
		requestPath = "/"
	}
	if basePath == "" || basePath == "/" {
		return requestPath
	}
	if requestPath == "/" {
		return strings.TrimRight(basePath, "/") + "/"
	}
	return strings.TrimRight(basePath, "/") + "/" + strings.TrimLeft(requestPath, "/")
}

func joinQuery(baseQuery, requestQuery string) string {
	if baseQuery == "" {
		return requestQuery
	}
	if requestQuery == "" {
		return baseQuery
	}
	return baseQuery + "&" + requestQuery
}

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func removeHopByHopHeaders(header http.Header) {
	connectionValues := append([]string(nil), header.Values("Connection")...)

	for _, key := range []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Proxy-Connection",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	} {
		header.Del(key)
	}

	for _, connectionValue := range connectionValues {
		for _, token := range strings.Split(connectionValue, ",") {
			header.Del(strings.TrimSpace(token))
		}
	}
}
