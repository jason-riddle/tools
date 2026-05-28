package protocol

import (
	"bytes"
	"encoding/gob"
	"testing"
)

// TestMessageGobRoundTrip verifies that the transport envelope survives gob encode/decode.
func TestMessageGobRoundTrip(t *testing.T) {
	want := Message{
		Version: 1,
		Type:    "ping",
		ID:      "123",
		Body:    []byte("hello"),
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(want); err != nil {
		t.Fatalf("Encode() unexpected error: %v", err)
	}

	var got Message
	if err := gob.NewDecoder(&buf).Decode(&got); err != nil {
		t.Fatalf("Decode() unexpected error: %v", err)
	}

	if got.Version != want.Version || got.Type != want.Type || got.ID != want.ID || !bytes.Equal(got.Body, want.Body) {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
}
