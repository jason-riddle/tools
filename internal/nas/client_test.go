package nas

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNormalizeBaseURLAddsHTTPSPort(t *testing.T) {
	got, err := normalizeBaseURL(Options{Host: "nas"})
	if err != nil {
		t.Fatalf("normalizeBaseURL() unexpected error: %v", err)
	}
	if got != "https://nas:5001" {
		t.Fatalf("normalizeBaseURL() = %q, want %q", got, "https://nas:5001")
	}
}

func TestLoginStatusAndLogout(t *testing.T) {
	var sawStatusToken bool
	var sawLogout bool

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		switch query.Get("api") {
		case "SYNO.API.Auth":
			switch query.Get("method") {
			case "login":
				fmt.Fprint(w, `{"success":true,"data":{"sid":"sid-1","synotoken":"token-1"}}`)
			case "logout":
				sawLogout = true
				if query.Get("_sid") != "sid-1" {
					t.Fatalf("logout sid = %q, want sid-1", query.Get("_sid"))
				}
				fmt.Fprint(w, `{"success":true}`)
			default:
				t.Fatalf("unexpected auth method %q", query.Get("method"))
			}
		case "SYNO.DSM.Info":
			if query.Get("_sid") != "sid-1" {
				t.Fatalf("status sid = %q, want sid-1", query.Get("_sid"))
			}
			if query.Get("SynoToken") != "token-1" {
				t.Fatalf("status SynoToken = %q, want token-1", query.Get("SynoToken"))
			}
			if r.Header.Get("X-SYNO-TOKEN") == "token-1" {
				sawStatusToken = true
			}
			fmt.Fprint(w, `{"success":true,"data":{"model":"DS220+","version_string":"DSM 7.3.2-86009 Update 3","uptime":71430}}`)
		case "SYNO.Core.System.Utilization":
			fmt.Fprint(w, `{"success":true,"data":{"cpu":{"user_load":4,"system_load":2},"memory":{"total_real":4096,"avail_real":2048}}}`)
		default:
			t.Fatalf("unexpected api %q", query.Get("api"))
		}
	}))
	defer server.Close()

	client, err := NewClient(Options{Timeout: 5 * time.Second, Insecure: true, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}

	if err := client.Login("jason", "secret"); err != nil {
		t.Fatalf("Login() unexpected error: %v", err)
	}

	status, err := client.Status()
	if err != nil {
		t.Fatalf("Status() unexpected error: %v", err)
	}
	if status.Model != "DS220+" {
		t.Fatalf("Status().Model = %q, want DS220+", status.Model)
	}
	if status.DSMVersion != "DSM 7.3.2-86009 Update 3" {
		t.Fatalf("Status().DSMVersion = %q", status.DSMVersion)
	}
	if status.CPUUser != 4 || status.CPUSystem != 2 {
		t.Fatalf("Status() cpu = user=%d sys=%d", status.CPUUser, status.CPUSystem)
	}
	if status.MemoryTotalKB != 4096 || status.MemoryAvailKB != 2048 {
		t.Fatalf("Status() memory = total=%d avail=%d", status.MemoryTotalKB, status.MemoryAvailKB)
	}
	if !sawStatusToken {
		t.Fatal("Status() did not set X-SYNO-TOKEN header")
	}

	if err := client.Logout(); err != nil {
		t.Fatalf("Logout() unexpected error: %v", err)
	}
	if !sawLogout {
		t.Fatal("Logout() did not call auth logout")
	}
}

func TestRebootReturnsAPIError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		switch query.Get("api") {
		case "SYNO.API.Auth":
			fmt.Fprint(w, `{"success":true,"data":{"sid":"sid-1","synotoken":"token-1"}}`)
		case "SYNO.Core.System":
			fmt.Fprint(w, `{"success":false,"error":{"code":105}}`)
		default:
			fmt.Fprint(w, `{"success":true}`)
		}
	}))
	defer server.Close()

	client, err := NewClient(Options{Timeout: 5 * time.Second, Insecure: true, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}
	if err := client.Login("jason", "secret"); err != nil {
		t.Fatalf("Login() unexpected error: %v", err)
	}

	err = client.Reboot()
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Reboot() error = %v, want APIError", err)
	}
	if apiErr.Code != 105 {
		t.Fatalf("Reboot() APIError code = %d, want 105", apiErr.Code)
	}
}

func TestLoginRequiresToken(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success":true,"data":{"sid":"sid-1"}}`)
	}))
	defer server.Close()

	client, err := NewClient(Options{Timeout: 5 * time.Second, Insecure: true, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}

	err = client.Login("jason", "secret")
	if err == nil || !strings.Contains(err.Error(), "missing synotoken") {
		t.Fatalf("Login() error = %v, want missing synotoken", err)
	}
}

func TestNewClientHonorsInsecureFalse(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success":true}`)
	}))
	defer server.Close()

	client, err := NewClient(Options{Timeout: time.Second, Insecure: false, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}

	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.httpClient.Transport)
	}
	if transport.TLSClientConfig == nil {
		t.Fatal("TLSClientConfig is nil")
	}
	if transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify = true, want false")
	}

	_, err = client.httpClient.Get(server.URL)
	if err == nil {
		t.Fatal("GET unexpectedly succeeded with self-signed cert")
	}
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		t.Fatalf("GET error = %v, want url.Error", err)
	}
	var certErr *tls.CertificateVerificationError
	if !errors.As(err, &certErr) {
		t.Fatalf("GET error = %v, want certificate verification error", err)
	}
	_ = server.Certificate()
	_ = transport
	_ = certErr
	_ = urlErr
}
