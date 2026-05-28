package client

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/jason-riddle/tools/internal/gob/protocol"
	serverpkg "github.com/jason-riddle/tools/internal/gob/server"
	"net/http/httptest"
)

func TestSendRoundTrip(t *testing.T) {
	ts := httptest.NewServer(serverpkg.Handler())
	defer ts.Close()

	msg := protocol.Message{
		Version: 1,
		Type:    "ping",
		ID:      "abc",
		Body:    []byte("hello"),
	}

	reply, err := Send(Options{Addr: strings.TrimPrefix(ts.URL, "http://"), Timeout: time.Second, Message: msg})
	if err != nil {
		t.Fatalf("Send() unexpected error: %v", err)
	}
	if reply.Version != msg.Version || reply.Type != msg.Type || reply.ID != msg.ID || !bytes.Equal(reply.Body, msg.Body) {
		t.Fatalf("Send() reply = %+v, want %+v", reply, msg)
	}
}
