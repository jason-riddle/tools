// Package protocol defines the gob envelope shared by the gob client and server.
package protocol

// Message is the gob envelope used for all gob communication.
//
// Version allows future protocol evolution without breaking existing decoders.
// Body remains raw bytes so the transport layer does not need gob type registration
// for arbitrary payload values.
type Message struct {
	Version uint8
	Type    string
	ID      string
	Body    []byte
}
