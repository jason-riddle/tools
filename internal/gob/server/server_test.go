package server

import (
	"bytes"
	"encoding/gob"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jason-riddle/tools/internal/gob/protocol"
)

func TestHandlerEchoesMessage(t *testing.T) {
	msg := protocol.Message{
		Version: 1,
		Type:    "ping",
		ID:      "123",
		Body:    []byte("hello"),
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(msg); err != nil {
		t.Fatalf("Encode() unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/send", &buf)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var reply protocol.Message
	if err := gob.NewDecoder(res.Body).Decode(&reply); err != nil {
		t.Fatalf("Decode() unexpected error: %v", err)
	}

	if reply.Version != msg.Version || reply.Type != msg.Type || reply.ID != msg.ID || !bytes.Equal(reply.Body, msg.Body) {
		t.Fatalf("reply = %+v, want %+v", reply, msg)
	}
}
