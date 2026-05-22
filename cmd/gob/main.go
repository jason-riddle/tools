package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jason-riddle/tools/internal/gob/client"
	"github.com/jason-riddle/tools/internal/gob/protocol"
	"github.com/jason-riddle/tools/internal/gob/server"
)

const usage = `gob - a gob message transport tool

Usage:
  gob <command> [flags]

Commands:
  server    Start the gob HTTP server
  client    Send a gob message to the server

Server flags:
  -listen string   address to listen on (default ":9000")

Client flags:
  -addr    string   server address        (default "localhost:9000")
  -id      string   message ID            (default "1")
  -type    string   message type          (default "ping")
  -body    string   message body          (default "hello")
  -timeout duration connection timeout    (default 5s)

Examples:
  gob server --listen :9000
  gob client --addr localhost:9000 --type ping --body "hello world"
`

func main() {
	log.SetFlags(0) // clean output, no timestamps from log prefix
	log.SetPrefix("gob: ")

	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "server":
		runServer(os.Args[2:])
	case "client":
		runClient(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
	default:
		fmt.Fprintf(os.Stderr, "gob: unknown command %q\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
}

// runServer parses server subcommand flags and starts the HTTP server.
func runServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	listen := fs.String("listen", ":9000", "address to listen on")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gob server [flags]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if err := server.Run(*listen); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// runClient parses client subcommand flags, builds a Message, and sends it.
func runClient(args []string) {
	fs := flag.NewFlagSet("client", flag.ExitOnError)
	addr := fs.String("addr", "localhost:9000", "server address")
	id := fs.String("id", "1", "message ID")
	msgType := fs.String("type", "ping", "message type (e.g. ping, chat)")
	body := fs.String("body", "hello", "message body")
	timeout := fs.Duration("timeout", 5*time.Second, "connection timeout")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gob client [flags]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	msg := protocol.Message{
		Version: 1,
		Type:    *msgType,
		ID:      *id,
		Body:    []byte(*body),
	}

	reply, err := client.Send(*addr, msg, *timeout)
	if err != nil {
		log.Fatalf("client error: %v", err)
	}

	fmt.Printf("sent    id=%s type=%s body=%q\n", msg.ID, msg.Type, msg.Body)
	fmt.Printf("replied id=%s type=%s body=%q\n", reply.ID, reply.Type, reply.Body)
}
