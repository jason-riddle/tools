package protocol

// Message is the gob envelope used for all goober communication.
// Version allows future protocol evolution without breaking existing decoders.
type Message struct {
	Version uint8
	Type    string
	ID      string
	Body    []byte
}
