# tools

A collection of Go tools. Currently contains `cfup`, an HTTP checker and proxy for services behind Cloudflare Access, `grab`, a single-file downloader, `gob`, a gob message transport tool over HTTP, `json`, a stable JSON formatter, `nas`, a Synology DSM CLI, `port`, a Portainer status CLI, `pub`, a GitHub public SSH key CLI, `tick`, a time CLI, and `uuid`, a UUID CLI.

## Install with Nix

Install all tools from the flake:

```bash
nix profile add github:jason-riddle/tools
```

Install a specific tool:

```bash
nix profile add github:jason-riddle/tools#gob
nix profile add github:jason-riddle/tools#grab
nix profile add github:jason-riddle/tools#json
nix profile add github:jason-riddle/tools#nas
nix profile add github:jason-riddle/tools#port
nix profile add github:jason-riddle/tools#pub
nix profile add github:jason-riddle/tools#tick
nix profile add github:jason-riddle/tools#uuid
nix profile add github:jason-riddle/tools#cfup
```

Build locally with Nix:

```bash
nix build 'path:.#default'
nix build 'path:.#gob'
nix build 'path:.#grab'
nix build 'path:.#json'
nix build 'path:.#nas'
nix build 'path:.#port'
nix build 'path:.#pub'
nix build 'path:.#tick'
nix build 'path:.#uuid'
nix build 'path:.#cfup'
```

## cfup

`cfup` checks whether an HTTP service behind Cloudflare Access is actually reachable, instead of treating the Access login page as a healthy upstream.

When `CF_ACCESS_CLIENT_ID` and `CF_ACCESS_CLIENT_SECRET` are both set, `cfup` injects them as `CF-Access-Client-Id` and `CF-Access-Client-Secret`.

### Build

```bash
go build -o cfup ./cmd/cfup
```

### Usage

```bash
./cfup https://example.com
./cfup --url https://example.com
./cfup --listen :8000 --url https://example.com
```

### Flags

| Flag | Description |
|------|-------------|
| `--url` | Target upstream URL |
| `--listen` | Listen address for proxy mode |
| `--timeout` | Per-request timeout (default `10s`) |
| `-h`, `-help`, `--help` | Print usage and examples |

### Check Mode

`cfup` follows redirects, evaluates the final response, and exits `0` for `200-399` and non-zero otherwise.

Example output:

```text
healthy status=200 final_url=https://example.com/app duration=182ms
```

### Proxy Mode

`cfup --listen :8000 --url https://example.com` starts an HTTP server that forwards incoming requests to the configured upstream base URL while preserving method, path, query string, and request body.

## grab

`grab` downloads a single file from a URL into the current directory.

If `-o` is not provided, it first tries the response `Content-Disposition` filename, then falls back to the last path segment of the final URL after redirects. It refuses to overwrite existing files unless `-f` is set.

### Build

```bash
go build -o grab ./cmd/grab
```

### Usage

```bash
./grab https://example.com/file.tar.gz
./grab -o archive.tar.gz https://example.com/file.tar.gz
./grab -f https://example.com/file.tar.gz
```

### Flags

| Flag | Description |
|------|-------------|
| `-o string` | Output filename |
| `-f` | Allow overwriting an existing file |
| `-timeout duration` | HTTP timeout (default `30s`) |
| `-h`, `-help`, `--help` | Print usage and examples |

## nas

`nas` interacts with a Synology NAS through the DSM Web API.

It reads connection details from the environment:

```bash
export NAS_HOST=nas
export NAS_USER=jason
export NAS_PASSWORD=secret
```

### Build

```bash
go build -o nas ./cmd/nas
```

### Usage

```bash
./nas status
./nas status -timeout 5s
./nas reboot
./nas reboot -confirm
```

### Flags

**status subcommand:**

| Flag | Default | Description |
|------|---------|-------------|
| `-timeout` | `10s` | Per-request HTTP timeout |
| `-insecure` | `true` | Skip TLS certificate verification |

**reboot subcommand:**

| Flag | Default | Description |
|------|---------|-------------|
| `-confirm` | `false` | Skip confirmation prompt |
| `-timeout` | `10s` | Per-request HTTP timeout |
| `-insecure` | `true` | Skip TLS certificate verification |

Example status output:

```text
Model:   DS220+
DSM:     DSM 7.3.2-86009 Update 3
Uptime:  14d 3h 22m
CPU:     user=4% sys=2%
Memory:  used=2048 MB / total=4096 MB
```

## port

`port` talks to the Portainer API and currently provides a `status` subcommand for checking server and endpoint health.

It reads connection details from flags or the environment:

```bash
export PORTAINER_URL=http://nas:9000
export PORTAINER_API_KEY=secret
```

### Build

```bash
go build -o port ./cmd/port
```

### Usage

```bash
./port status
./port status --url http://nas:9000 --api-key secret
./port status --json
```

### Flags

**status subcommand:**

| Flag | Default | Description |
|------|---------|-------------|
| `--url` | env | Portainer base URL |
| `--api-key` | env | Portainer API key sent as `X-API-Key` |
| `--timeout` | `10s` | Per-request HTTP timeout |
| `--json` | `false` | Emit JSON output |

Example status output:

```text
Portainer: http://nas:9000
Version:   2.33.4
Auth:      ok
Endpoints: 1

local (id=2, type=docker, status=up)
  URL:          unix:///var/run/docker.sock
  Public URL:   http://nas
  Snapshot:     2026-05-28T07:45:04Z
  Docker:       24.0.2
  Containers:   total=12 running=10 stopped=2
  Health:       healthy=0 unhealthy=0
  Images:       15
  Volumes:      111
  Stacks:       8
  Services:     0
```

## json

`json` pretty-prints JSON with stable, sorted object keys.

By default it preserves array order and sorts object keys at every depth.
Three flags give finer control over how deep sorting applies:

- `--sort-arrays` sorts arrays of scalar values; use `--arrays-depth N` to limit how many array levels deep that goes.
- `--sort-keys-min-depth N` sets the first object level at which key sorting begins (default 1 = top level).
- `--sort-keys-max-depth N` sets the last object level at which key sorting applies (default -1 = unlimited).

Use both min/max together to sort keys at exactly one depth — for example, `--sort-keys-min-depth 2 --sort-keys-max-depth 2` leaves top-level keys in input order while sorting the next level down.

### Build

```bash
go build -o json ./cmd/json
```

### Usage

```bash
./json < file.json
./json file.json
./json --compact file.json
./json --sort-arrays file.json
./json --sort-arrays --arrays-depth 1 file.json
./json --sort-keys-min-depth 1 --sort-keys-max-depth 1 file.json
./json --sort-keys-min-depth 2 --sort-keys-max-depth 2 file.json
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--sort-arrays` | `false` | Sort arrays of scalar values recursively |
| `--arrays-depth N` | `-1` | Limit array sorting to N array levels deep (`-1` = unlimited). Requires `--sort-arrays`. |
| `--sort-keys-min-depth N` | `1` | First object level at which key sorting begins. `1` = top level. |
| `--sort-keys-max-depth N` | `-1` | Last object level at which key sorting applies. `-1` = no upper bound. |
| `--compact` | `false` | Emit compact JSON instead of pretty-printed output |
| `-h`, `-help`, `--help` | | Print usage and examples |

### Examples

**Default — pretty-print with sorted keys at all levels:**

```
Input:  {"z":1,"a":2}
Output: {
          "a": 2,
          "z": 1
        }
```

**`--sort-arrays` — sort scalar arrays everywhere:**

```
Input:  {"tags":["z","a","m"],"ids":[3,1,2]}
Output: {
          "ids": [1, 2, 3],
          "tags": ["a", "m", "z"]
        }
```

**`--sort-arrays --arrays-depth 1` — sort only the first array level; arrays inside arrays are left alone:**

```
Input:  {"top":[3,1,2],"matrix":[[5,4],[8,6]]}
Output: {
          "matrix": [[5,4],[8,6]],
          "top": [1, 2, 3]
        }
```

**`--sort-keys-min-depth 1 --sort-keys-max-depth 1` — sort only top-level object keys; nested keys keep input order:**

```
Input:  {"z":{"y":1,"x":2},"b":{"d":3,"c":4}}
Output: {
          "b": {
            "d": 3,
            "c": 4
          },
          "z": {
            "y": 1,
            "x": 2
          }
        }
```

**`--sort-keys-min-depth 2 --sort-keys-max-depth 2` — leave top-level keys in input order, sort only depth-2 keys:**

This is useful for files like a skill lock file where `version` must stay first but the skill names inside `skills` should be alphabetical.

```
Input:  {"version":3,"skills":{"z-skill":{},"a-skill":{}}}
Output: {
          "version": 3,
          "skills": {
            "a-skill": {},
            "z-skill": {}
          }
        }
```

## pub

`pub` prints the public SSH keys for a GitHub user by fetching `https://github.com/<user>.keys`.

### Build

```bash
go build -o pub ./cmd/pub
```

### Usage

```bash
./pub
./pub octocat
./pub -timeout 5s octocat
```

### Flags

| Flag | Description |
|------|-------------|
| `-timeout duration` | HTTP timeout (default `10s`) |
| `-h`, `-help`, `--help` | Print usage and examples |

## tick

`tick` prints the current time.

By default it prints an RFC 3339 timestamp in UTC. If `TZ` is set, `tick` loads that location and formats the output in that timezone.

### Build

```bash
go build -o tick ./cmd/tick
```

### Usage

```bash
./tick
./tick +24h
./tick -nano
./tick -epoch
./tick -format '2006-01-02 15:04:05 MST'
./tick -json
TZ=America/New_York ./tick
```

### Flags

| Flag | Description |
|------|-------------|
| `-nano` | Print in RFC3339Nano format |
| `-epoch` | Print Unix epoch seconds |
| `-format string` | Print using a Go time layout string |
| `-json` | Print common `time` package layouts as JSON |

`-nano`, `-epoch`, `-format`, and `-json` are mutually exclusive.

`tick` accepts one optional positional duration offset, such as `+24h` or `+30s`. The offset must appear after any flags.

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
./gob server -listen :9000
```

**Send a message:**

```bash
./gob client -addr localhost:9000 -type ping -body "hello world"
./gob client -addr localhost:9000 -type chat -body "hey server" -id abc123
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
