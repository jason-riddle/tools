package port

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStatusAndEndpointsUseAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-Key"); got != "secret-key" {
			t.Fatalf("X-API-Key = %q, want %q", got, "secret-key")
		}

		switch r.URL.Path {
		case "/api/status":
			fmt.Fprint(w, `{"Version":"2.33.4","InstanceID":"instance-1"}`)
		case "/api/endpoints":
			fmt.Fprint(w, `[{"Id":2,"Name":"local","Type":1,"URL":"unix:///var/run/docker.sock","PublicURL":"http://nas","Status":1,"Snapshots":[{"Time":1710000000,"DockerVersion":"24.0.2","TotalCPU":2,"TotalMemory":1024,"ContainerCount":12,"RunningContainerCount":10,"StoppedContainerCount":2,"HealthyContainerCount":0,"UnhealthyContainerCount":0,"VolumeCount":111,"ImageCount":15,"ServiceCount":0,"StackCount":8}]}]`)
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(Options{BaseURL: server.URL, APIKey: "secret-key", Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}

	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() unexpected error: %v", err)
	}
	if status.Version != "2.33.4" {
		t.Fatalf("Status().Version = %q, want %q", status.Version, "2.33.4")
	}

	endpoints, err := client.Endpoints(context.Background())
	if err != nil {
		t.Fatalf("Endpoints() unexpected error: %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("len(Endpoints()) = %d, want 1", len(endpoints))
	}
	if endpoints[0].Name != "local" {
		t.Fatalf("Endpoints()[0].Name = %q, want %q", endpoints[0].Name, "local")
	}
	if len(endpoints[0].Snapshots) != 1 {
		t.Fatalf("len(Endpoints()[0].Snapshots) = %d, want 1", len(endpoints[0].Snapshots))
	}
}

func TestEndpointsReturnsAPIErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"message":"Invalid JWT token","details":"Unauthorized"}`)
	}))
	defer server.Close()

	client, err := NewClient(Options{BaseURL: server.URL, APIKey: "wrong", Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}

	_, err = client.Endpoints(context.Background())
	if err == nil {
		t.Fatal("Endpoints() expected an error")
	}
	if !strings.Contains(err.Error(), "Invalid JWT token") {
		t.Fatalf("Endpoints() error = %q, want API message", err)
	}
}

func TestNewClientRejectsMissingConfig(t *testing.T) {
	_, err := NewClient(Options{APIKey: "secret"})
	if err == nil || !strings.Contains(err.Error(), "missing Portainer base URL") {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = NewClient(Options{BaseURL: "http://nas:9000"})
	if err == nil || !strings.Contains(err.Error(), "missing Portainer API key") {
		t.Fatalf("NewClient() error = %v", err)
	}
}
