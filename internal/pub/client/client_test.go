package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchReturnsExactBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/octocat.keys" {
			t.Fatalf("request path = %q, want %q", r.URL.Path, "/octocat.keys")
		}
		_, _ = io.WriteString(w, "ssh-ed25519 AAAATEST octocat\n")
	}))
	defer server.Close()

	body, err := Fetch("octocat", Options{BaseURL: server.URL, Timeout: time.Second})
	if err != nil {
		t.Fatalf("Fetch() unexpected error: %v", err)
	}

	if got := string(body); got != "ssh-ed25519 AAAATEST octocat\n" {
		t.Fatalf("Fetch() body = %q", got)
	}
}

func TestFetchRejectsNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	}))
	defer server.Close()

	_, err := Fetch("missing", Options{BaseURL: server.URL, Timeout: time.Second})
	if err == nil {
		t.Fatal("Fetch() expected an error")
	}
	if !strings.Contains(err.Error(), "404 Not Found") {
		t.Fatalf("Fetch() error = %q, want status text", err)
	}
}
