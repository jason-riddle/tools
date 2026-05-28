package grab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDownloadUsesContentDispositionFilename(t *testing.T) {
	t.Setenv("PWD", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="archive.tar.gz"`)
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	wd := mustGetwd(t)
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	name, err := Download(context.Background(), &http.Client{Timeout: time.Second}, server.URL, Options{})
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if name != "archive.tar.gz" {
		t.Fatalf("Download() name = %q, want %q", name, "archive.tar.gz")
	}
	data, err := os.ReadFile(filepath.Join(".", name))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "content" {
		t.Fatalf("saved content = %q, want %q", string(data), "content")
	}
}

func TestDownloadUsesFinalRedirectURLFilename(t *testing.T) {
	wd := mustGetwd(t)
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, "/files/report.txt", http.StatusFound)
		case "/files/report.txt":
			_, _ = w.Write([]byte("report"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	name, err := Download(context.Background(), &http.Client{Timeout: time.Second}, server.URL+"/start", Options{})
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if name != "report.txt" {
		t.Fatalf("Download() name = %q, want %q", name, "report.txt")
	}
}

func TestDownloadRequiresFilenameWhenNoneCanBeDetected(t *testing.T) {
	wd := mustGetwd(t)
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	_, err := Download(context.Background(), &http.Client{Timeout: time.Second}, server.URL, Options{})
	if err == nil || !strings.Contains(err.Error(), "use -o") {
		t.Fatalf("Download() error = %v, want message containing %q", err, "use -o")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("ReadDir() entries = %d, want 0", len(entries))
	}
}

func TestDownloadRefusesOverwriteWithoutForce(t *testing.T) {
	wd := mustGetwd(t)
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	if err := os.WriteFile("file.txt", []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("new"))
	}))
	defer server.Close()

	_, err := Download(context.Background(), &http.Client{Timeout: time.Second}, server.URL, Options{Output: "file.txt"})
	if err == nil || !strings.Contains(err.Error(), `refusing to overwrite existing file "file.txt"`) {
		t.Fatalf("Download() error = %v, want overwrite refusal", err)
	}
	data, err := os.ReadFile("file.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "old" {
		t.Fatalf("saved content = %q, want %q", string(data), "old")
	}
}

func TestDownloadOverwritesWithForce(t *testing.T) {
	wd := mustGetwd(t)
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	if err := os.WriteFile("file.txt", []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("new"))
	}))
	defer server.Close()

	name, err := Download(context.Background(), &http.Client{Timeout: time.Second}, server.URL, Options{Output: "file.txt", Force: true})
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if name != "file.txt" {
		t.Fatalf("Download() name = %q, want %q", name, "file.txt")
	}
	data, err := os.ReadFile("file.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "new" {
		t.Fatalf("saved content = %q, want %q", string(data), "new")
	}
}

func TestDownloadRejectsNon200Responses(t *testing.T) {
	wd := mustGetwd(t)
	dir := t.TempDir()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	}))
	defer server.Close()

	_, err := Download(context.Background(), &http.Client{Timeout: time.Second}, server.URL+"/missing.txt", Options{})
	if err == nil || !strings.Contains(err.Error(), "404 Not Found") {
		t.Fatalf("Download() error = %v, want 404 status", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("ReadDir() entries = %d, want 0", len(entries))
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	return wd
}
