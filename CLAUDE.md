# CLAUDE.md — Development Guidelines for copilot-sync

## Project Overview

`copilot-sync` (`cops`) is a deterministic package manager for GitHub Copilot agent files.
It manages instructions, agents, prompts, and skills from GitHub repositories through a `copilot.toml` manifest.

## Quick Reference

```bash
# Run tests (required before every commit)
go test -race ./...

# Run tests with coverage
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out

# Build
go build ./...

# Lint (if golangci-lint is installed)
golangci-lint run ./...
```

## Architecture

```
cmd/cops/main.go          → Entry point (calls cli.Execute)
internal/
  auth/                   → GitHub token auth (env vars: GITHUB_TOKEN, GH_TOKEN)
  cli/                    → Cobra CLI commands (use, unuse, sync, check)
  config/                 → Asset types (instructions/agents/prompts/skills) and ref parsing
  injector/               → Downloads + writes assets to .github/<type>/ directories
  manifest/               → copilot.toml (TOML) and .cops.lock (JSON) file management
  resolver/               → GitHub API client (raw content + trees + commits)
```

## Key Invariants — Do Not Break

1. **Deterministic output**: TOML manifest saves with sorted keys. Lock file JSON is indented. Directory checksums are computed from sorted file paths.
2. **Lock file format**: `.cops.lock` is JSON with `version: 1`. Entries keyed by `<type>/<name>`. Do not change the schema without a migration plan.
3. **Asset type conventions**: Instructions → `.instructions.md`, Agents → `.agent.md`, Prompts → `.prompt.md`, Skills → directories.
4. **ResolverAPI interface**: The `resolver.ResolverAPI` interface enables testing without GitHub. Always use the interface in Injector and CLI commands, never the concrete `*Resolver` directly.

## Testing Requirements

- **All new code must have tests.** Use table-driven tests and `t.Parallel()`.
- **CLI commands must be testable**: Use the `run*With()` pattern — the command runner accepts dependencies (paths, resolver) as parameters. The Cobra handler is a thin wrapper.
- **Resolver tests**: Use `net/http/httptest` with the `rewriteTransport` pattern (see `resolver_test.go`).
- **Mock resolver**: Use `mockResolver` from `cli_test.go` for CLI integration tests.
- **Coverage threshold**: CI fails if total coverage drops below 45%.
- **Run `go test -race ./...` before committing.**

## Dependency Policy

- Minimize external dependencies. Currently only 2: `BurntSushi/toml` and `spf13/cobra`.
- Prefer stdlib solutions over third-party packages.
- No test frameworks — use `testing` package only.

## CI Pipeline

GitHub Actions (`.github/workflows/ci.yml`) runs on push to main and PRs:
1. golangci-lint (errcheck, govet, staticcheck, unused, ineffassign)
2. `go test -race` with coverage
3. Coverage threshold check (45% minimum)
