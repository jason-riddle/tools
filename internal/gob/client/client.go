package client

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jason-riddle/tools/internal/gob/protocol"
)

// Send gob-encodes msg and POSTs it to addr/send, then decodes the echoed response.
func Send(addr string, msg protocol.Message, timeout time.Duration) (protocol.Message, error) {
	var buf bytes.Buffer

	log.Printf("send  version=%d type=%q id=%q addr=%s body=%q",
		msg.Version, msg.Type, msg.ID, addr, msg.Body)

	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(msg); err != nil {
		return protocol.Message{}, fmt.Errorf("encode: %w", err)
	}

	log.Printf("post  url=http://%s/send bytes=%d", addr, buf.Len())

	httpClient := &http.Client{Timeout: timeout}
	resp, err := httpClient.Post("http://"+addr+"/send", "application/octet-stream", &buf)
	if err != nil {
		return protocol.Message{}, fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("resp  status=%s", resp.Status)

	if resp.StatusCode != http.StatusOK {
		return protocol.Message{}, fmt.Errorf("server returned %s", resp.Status)
	}

	var reply protocol.Message
	dec := gob.NewDecoder(resp.Body)
	if err := dec.Decode(&reply); err != nil {
		return protocol.Message{}, fmt.Errorf("decode response: %w", err)
	}

	log.Printf("reply version=%d type=%q id=%q body=%q",
		reply.Version, reply.Type, reply.ID, reply.Body)

	return reply, nil
}
