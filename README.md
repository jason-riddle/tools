# tools

A collection of Go tools. Currently contains `gob`, a gob message transport tool over HTTP, and `uuid`, a UUID CLI.

## Install with Nix

Install both tools from the flake:

```bash
nix profile add github:jason-riddle/tools
```

Install a specific tool:

```bash
nix profile add github:jason-riddle/tools#gob
nix profile add github:jason-riddle/tools#uuid
```

Build locally with Nix:

```bash
nix build 'path:.#default'
nix build 'path:.#gob'
nix build 'path:.#uuid'
```

## gob

`gob` sends and receives [gob](https://pkg.go.dev/encoding/gob)-encoded messages over HTTP. It runs as either a server or a client from the same binary.

### Directory structure

```
cmd/
  gob/
    main.go               CLI entrypoint (server + client subcommands)
internal/
  gob/
    protocol/
      message.go          Message struct (gob envelope)
    server/
      server.go           HTTP server + gob handler
    client/
      client.go           HTTP client sending gob payloads
```

### Build

```bash
go build -o gob ./cmd/gob
```

### Usage

**Start the server:**

```bash
./gob server --listen :9000
```

**Send a message:**

```bash
./gob client --addr localhost:9000 --type ping --body "hello world"
./gob client --addr localhost:9000 --type chat --body "hey server" --id abc123
```

**Expected output:**

Server terminal:
```
gob: listening on :9000
gob: recv  version=1 type="ping" id="1" from=127.0.0.1:54321 body="hello world"
```

Client terminal:
```
gob: send  version=1 type="ping" id="1" addr=localhost:9000 body="hello world"
gob: post  url=http://localhost:9000/send bytes=85
gob: resp  status=200 OK
gob: reply version=1 type="ping" id="1" body="hello world"
sent    id=1 type=ping body="hello world"
replied id=1 type=ping body="hello world"
```

### Flags

**server subcommand:**

| Flag | Default | Description |
|------|---------|-------------|
| `-listen` | `:9000` | Address to listen on |

**client subcommand:**

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `localhost:9000` | Server address |
| `-id` | `1` | Message ID |
| `-type` | `ping` | Message type |
| `-body` | `hello` | Message body |
| `-timeout` | `5s` | Connection timeout |

### Design

| Decision | Rationale |
|---|---|
| One `go.mod` | Single module — all `cmd/` and `internal/` share the same module path |
| `flag.FlagSet` per subcommand | stdlib only; each subcommand gets its own isolated flag set, no global state |
| HTTP transport (not raw TCP) | Easier to test with `curl`, proxies, and standard tooling; gob is still the wire format in the body |
| `internal/` packages | Cannot be imported by other modules — enforces encapsulation |
| `[]byte` body, not `any` | Avoids `gob.Register` complexity; transport layer stays clean and versionable |
| `Version uint8` in envelope | Future-proofs the protocol; decoder can switch on version before acting |

## uuid

`uuid` generates and inspects UUIDs.

### Build

```bash
go build -o uuid ./cmd/uuid
```

### Usage

```bash
./uuid new
./uuid new -v 7
./uuid parse f81d4fae-7dec-11d0-a765-00a0c91e6bf6
./uuid version f81d4fae-7dec-11d0-a765-00a0c91e6bf6
```
