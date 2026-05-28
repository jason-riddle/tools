// Package server serves gob protocol messages over HTTP.
package server

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/http"

	"github.com/jason-riddle/tools/internal/gob/protocol"
)

// Options configures the gob HTTP server.
type Options struct {
	Listen string
}

// Handler returns an HTTP handler that accepts gob-encoded Messages on POST /send.
func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /send", handleSend)
	return mux
}

func handleSend(w http.ResponseWriter, r *http.Request) {
	var msg protocol.Message

	dec := gob.NewDecoder(r.Body)
	if err := dec.Decode(&msg); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode gob: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("recv  version=%d type=%q id=%q from=%s body=%q",
		msg.Version, msg.Type, msg.ID, r.RemoteAddr, msg.Body)

	// Echo the message back as a gob response.
	w.Header().Set("Content-Type", "application/octet-stream")
	enc := gob.NewEncoder(w)
	if err := enc.Encode(msg); err != nil {
		log.Printf("error encoding response: %v", err)
	}
}

// Run starts the gob HTTP server on opts.Listen.
func Run(opts Options) error {
	log.Printf("listening on %s", opts.Listen)
	return http.ListenAndServe(opts.Listen, Handler())
}
