// Package client sends gob protocol messages over HTTP.
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

// Options configures a gob client request.
//
// Addr is the host:port of the gob server. Timeout is applied to the underlying
// HTTP client. Message is the gob envelope to send.
type Options struct {
	Addr    string
	Timeout time.Duration
	Message protocol.Message
}

// Send gob-encodes opts.Message and POSTs it to http://opts.Addr/send.
//
// It expects an HTTP 200 response containing a gob-encoded reply message.
func Send(opts Options) (protocol.Message, error) {
	var buf bytes.Buffer

	log.Printf("send  version=%d type=%q id=%q addr=%s body=%q",
		opts.Message.Version, opts.Message.Type, opts.Message.ID, opts.Addr, opts.Message.Body)

	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(opts.Message); err != nil {
		return protocol.Message{}, fmt.Errorf("encode: %w", err)
	}

	log.Printf("post  url=http://%s/send bytes=%d", opts.Addr, buf.Len())

	httpClient := &http.Client{Timeout: opts.Timeout}
	resp, err := httpClient.Post("http://"+opts.Addr+"/send", "application/octet-stream", &buf)
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
