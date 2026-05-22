# AGENTS

Guidelines for AI agents working in this repository.

## Repository overview

This is a single Go module (`github.com/jason-riddle/tools`) containing CLI tools under `cmd/` and shared packages under `internal/`.

- **One `go.mod`** at the repo root. Do not create additional `go.mod` files unless explicitly required for a separate module.
- **No external dependencies** — stdlib only. Do not add third-party imports unless explicitly requested.
- All packages under `internal/` are intentionally unexported outside the module.

## Tools

### goober

`cmd/goober` — a gob-over-HTTP message transport tool.

Key packages:
- `internal/goober/protocol` — `Message` struct (the gob envelope)
- `internal/goober/server` — HTTP server that decodes incoming gob messages and echoes them back
- `internal/goober/client` — HTTP client that encodes and POSTs gob messages

## Development workflow

**Build:**
```bash
go build ./...
```

**Vet:**
```bash
go vet ./...
```

**Test:**
```bash
go test ./...
```

**Build a specific binary:**
```bash
go build -o goober ./cmd/goober
```

Always run `go build ./...` and `go vet ./...` after making changes to verify nothing is broken.

## Code conventions

- Use `flag.FlagSet` per subcommand (not global `flag` package functions) to avoid shared state between subcommands.
- Log output uses `log.SetPrefix` and `log.SetFlags(0)` for clean prefixed output without timestamps.
- Error messages from `log.Fatalf` are prefixed automatically by the log package.
- Prefer `fmt.Errorf("context: %w", err)` for error wrapping.
- Keep `internal/protocol.Message.Body` as `[]byte` — do not change it to `any` or `interface{}` as that requires `gob.Register` calls.

## Adding a new tool

1. Create `cmd/<toolname>/main.go` with `package main`.
2. Add shared packages under `internal/<toolname>/` if needed.
3. Import using the full module path: `github.com/jason-riddle/tools/internal/<toolname>/<pkg>`.
4. Update `README.md` with usage documentation.
5. Run `go build ./...` and `go vet ./...` to verify.
