# Contributing

Thanks for taking the time to contribute.

This repository contains small Go command-line tools built as a single module. Contributions should keep the codebase simple, standard-library-first, and easy to maintain.

## Before You Start

- Read `README.md` for the current tools, usage examples, and project layout.
- Check whether your change affects documentation, Nix packaging, or multiple tools.
- Keep changes focused. Small, reviewable pull requests are preferred over large mixed changes.

## Repository Layout

This repository is a single Go module: `github.com/jason-riddle/tools`.

- `cmd/` contains CLI entry points.
- `internal/` contains shared implementation packages.
- `README.md` documents the tools and their usage.
- `flake.nix` defines Nix packages for installable tools.

Repository conventions:

- Keep one `go.mod` at the repository root.
- Do not add third-party dependencies unless there is a clear, explicit reason to do so.
- Packages under `internal/` are intended for use only within this module.

## Development Setup

The project uses Go 1.22.

Common commands:

```bash
go build ./...
go vet ./...
go test ./...
```

Make targets are also available:

```bash
make build-all
make vet
make test
```

To build with Nix:

```bash
nix build 'path:.#default'
nix build 'path:.#gob'
nix build 'path:.#uuid'
```

## Making Changes

When contributing code:

- Prefer small, direct changes over broad refactors.
- Keep tools independent unless code is clearly shared.
- Follow the existing project style and standard library usage.
- Update documentation when behavior, flags, or commands change.

If you add or modify a command-line interface, make sure the usage shown in `README.md` stays accurate.

## Code Style

This project intentionally keeps the code straightforward.

- Use `flag.FlagSet` per subcommand instead of the global `flag` package state.
- Use `log.SetPrefix` and `log.SetFlags(0)` for clean CLI log output.
- Prefer `fmt.Errorf("context: %w", err)` for wrapping errors.
- Keep transport and protocol types simple and explicit.

For the gob tool specifically:

- Keep `internal/gob/protocol.Message.Body` as `[]byte`.
- Do not change it to `any` or `interface{}` unless the design is intentionally being revised, since that would require additional `gob.Register` handling.

## Adding a New Tool

To add a new CLI tool:

1. Create `cmd/<toolname>/main.go` with `package main`.
2. Add any shared implementation under `internal/<toolname>/`.
3. Import internal packages using the full module path.
4. Document the tool in `README.md`.
5. If the tool should be installable through Nix, add it to `flake.nix`.
6. Run the verification commands before submitting the change.

## Verification

Before opening a pull request, run at least:

```bash
go build ./...
go vet ./...
go test ./...
```

If you change `flake.nix`, also build the affected flake outputs.

## Pull Requests

Pull requests are easier to review when they:

- describe the problem being solved
- explain the approach briefly
- keep unrelated changes out of the diff
- include documentation updates when needed

If there is a tradeoff or design choice that is not obvious from the code, call it out in the pull request description.
