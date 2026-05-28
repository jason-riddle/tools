package cfup

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCheckFollowsRedirectsAndInjectsAccessHeaders(t *testing.T) {
	targetToken := AccessToken{ClientID: "client-id", ClientSecret: "client-secret"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			if got := r.Header.Get("CF-Access-Client-Id"); got != targetToken.ClientID {
				t.Fatalf("CF-Access-Client-Id = %q, want %q", got, targetToken.ClientID)
			}
			if got := r.Header.Get("CF-Access-Client-Secret"); got != targetToken.ClientSecret {
				t.Fatalf("CF-Access-Client-Secret = %q, want %q", got, targetToken.ClientSecret)
			}
			http.Redirect(w, r, "/accounts/login/?next=/", http.StatusFound)
		case "/accounts/login/":
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "Paperless-ngx sign in")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	result, err := Check(context.Background(), &http.Client{Timeout: time.Second}, target, targetToken)
	if err != nil {
		t.Fatalf("Check() unexpected error: %v", err)
	}
	if !result.Healthy {
		t.Fatalf("Check() Healthy = false, want true")
	}
	if !result.ReachedApp {
		t.Fatalf("Check() ReachedApp = false, want true")
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("Check() status = %d, want %d", result.StatusCode, http.StatusOK)
	}
	if !strings.HasSuffix(result.FinalURL, "/accounts/login/?next=/") {
		t.Fatalf("Check() final URL = %q, want redirect target", result.FinalURL)
	}
	if result.Duration <= 0 {
		t.Fatalf("Check() duration = %s, want > 0", result.Duration)
	}
}

func TestCheckMarksUnhealthyStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer server.Close()

	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	result, err := Check(context.Background(), &http.Client{Timeout: time.Second}, target, AccessToken{})
	if err != nil {
		t.Fatalf("Check() unexpected error: %v", err)
	}
	if result.Healthy {
		t.Fatalf("Check() Healthy = true, want false")
	}
	if !result.ReachedApp {
		t.Fatalf("Check() ReachedApp = false, want true")
	}
	if result.StatusCode != http.StatusBadGateway {
		t.Fatalf("Check() status = %d, want %d", result.StatusCode, http.StatusBadGateway)
	}
	if result.FinalURL != server.URL {
		t.Fatalf("Check() final URL = %q, want %q", result.FinalURL, server.URL)
	}
}

func TestCheckMarksCloudflareAccessLoginUnhealthy(t *testing.T) {
	const protectedHost = "paperless-v2.jasonriddle.com"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cf-Access-Domain", protectedHost)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "Sign in · Cloudflare Access")
	}))
	defer server.Close()

	target, err := url.Parse("https://" + protectedHost)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	client := server.Client()
	client.Timeout = time.Second

	result, err := Check(context.Background(), client, target, AccessToken{})
	if err != nil {
		t.Fatalf("Check() unexpected error: %v", err)
	}
	if result.Healthy {
		t.Fatalf("Check() Healthy = true, want false")
	}
	if result.ReachedApp {
		t.Fatalf("Check() ReachedApp = true, want false")
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("Check() status = %d, want %d", result.StatusCode, http.StatusOK)
	}
}

func TestResolveProxyURLPreservesBasePathAndQuery(t *testing.T) {
	base, err := url.Parse("https://example.com/app?mode=proxy")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	incoming := &url.URL{Path: "/api/documents", RawQuery: "page=2"}

	resolved := ResolveProxyURL(base, incoming)
	if got, want := resolved.String(), "https://example.com/app/api/documents?mode=proxy&page=2"; got != want {
		t.Fatalf("ResolveProxyURL() = %q, want %q", got, want)
	}
}

func TestProxyHandlerForwardsRequestAndInjectsAccessHeaders(t *testing.T) {
	targetToken := AccessToken{ClientID: "client-id", ClientSecret: "client-secret"}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodPost; got != want {
			t.Fatalf("method = %q, want %q", got, want)
		}
		if got, want := r.URL.Path, "/base/upload"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		if got, want := r.URL.RawQuery, "debug=1&q=test"; got != want {
			t.Fatalf("raw query = %q, want %q", got, want)
		}
		if got := r.Header.Get("X-Test"); got != "keep-me" {
			t.Fatalf("X-Test = %q, want keep-me", got)
		}
		if got := r.Header.Get("CF-Access-Client-Id"); got != targetToken.ClientID {
			t.Fatalf("CF-Access-Client-Id = %q, want %q", got, targetToken.ClientID)
		}
		if got := r.Header.Get("CF-Access-Client-Secret"); got != targetToken.ClientSecret {
			t.Fatalf("CF-Access-Client-Secret = %q, want %q", got, targetToken.ClientSecret)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		if got := string(body); got != "hello upstream" {
			t.Fatalf("body = %q, want %q", got, "hello upstream")
		}
		w.Header().Set("X-Upstream", "ok")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "proxied")
	}))
	defer upstream.Close()

	base, err := url.Parse(upstream.URL + "/base?debug=1")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	proxy := httptest.NewServer(NewProxyHandler(base, &http.Client{
		Timeout: time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, targetToken))
	defer proxy.Close()

	req, err := http.NewRequest(http.MethodPost, proxy.URL+"/upload?q=test", strings.NewReader("hello upstream"))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	req.Header.Set("X-Test", "keep-me")
	req.Header.Set("Connection", "keep-alive")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if got, want := resp.StatusCode, http.StatusCreated; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if got := resp.Header.Get("X-Upstream"); got != "ok" {
		t.Fatalf("X-Upstream = %q, want ok", got)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if got := string(body); got != "proxied" {
		t.Fatalf("body = %q, want proxied", got)
	}
}
