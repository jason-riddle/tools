# AGENTS

Guidance for AI agents working in this repository.

## Read First

Read `CONTRIBUTING.md` before making changes. It contains the shared repository guidance that applies to both human and agent contributors.

## Agent-Specific Expectations

- Keep changes narrow and task-focused.
- Prefer small edits over broad refactors.
- Do not add third-party dependencies unless explicitly requested.
- Do not create additional `go.mod` files unless the task specifically requires a separate module.
- When changing behavior, update `README.md` or other relevant documentation if needed.

## Verification

After making code changes, run:

```bash
go build ./...
go vet ./...
```

Also run:

```bash
go test ./...
```

If `flake.nix` changes, verify the affected Nix build targets as well.

## Repository Notes

- This repository is a single Go module: `github.com/jason-riddle/tools`.
- CLI entry points live under `cmd/`.
- Shared implementation packages live under `internal/`.
- Internal packages should remain internal to this module.
