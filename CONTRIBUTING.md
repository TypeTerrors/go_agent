# Contributing / Developer Guide

This project is a Go CLI with a small Makefile. If you’re coming from Node.js, think of the Makefile targets like npm scripts.

## Prerequisites

- Recommended: Go 1.22+
- Optional: gum (pretty terminal output), staticcheck, goimports
- Optional (for demos): vhs

If you don’t have Go or package managers installed, the Makefile will try reasonable fallbacks and print next steps.

## Quick Start

- Clone the repo
- Install tools (optional but nice):
  - make tools
- Build the CLI:
  - make build
- Run the CLI:
  - make run RUN_ARGS='--help'

## Common Tasks

- Format: make fmt (gofmt + goimports)
- Lint: make lint (go vet + staticcheck)
- Test: make test (all), make testv (verbose tests pkg), make test-one TEST_RUN='^TestName$'
- Race tests: make race
- Run commands (tool): the agent can execute shell with permissions via run_command
  - Read-only examples (permissions=r): git status, go env, go build (may require rw if toolchain writes cache)
  - Write examples (permissions=rw): git add/commit, go mod tidy
  - Execute-by-path (permissions=rx): ./bin/tool, ./script.sh
- Tidy modules: make tidy

If you prefer pure go commands (no Makefile):
- Build: go build -o bin/agent ./cmd/agent
- Run:  ./bin/agent --help
- Test: go test ./...
- Lint: go vet ./...; staticcheck ./... (if installed)
- Format: gofmt -w .; goimports -w .

## Platform Notes (macOS / Linux / Windows)

Tools install heuristics (make tools):
- With Go: installs via `go install`
- macOS: uses `brew` if present; otherwise suggests installing Go
- Linux: tries apt/dnf/yum/pacman; otherwise suggests installing Go
- Windows: tries `scoop` or `choco`; otherwise suggests installing Go or Scoop

You can always install Go from https://go.dev/dl/ and re-run `make tools`.

## Repo Structure

- cmd/agent: CLI entry point
- internal/services/agent: core agent logic (Run loop, planning, tooling)
- internal/services/prompts: embedded prompt text
- pkg: shared utilities (config, locks, logging, tool call helpers)
- tests: integration-style tests that exercise the tools

## Coding Style

- Use gofmt/goimports (make fmt)
- Keep imports grouped: stdlib, third-party, local
- Return (val, error); don’t panic in library code
- Wrap errors: fmt.Errorf("context: %w", err)
- Use pkg/logger for logs in libraries
- Prefer context + x/sync/errgroup for concurrency where applicable

## Running the Agent

Examples:
- Basic help: make run RUN_ARGS='--help'
- Run with task: make run RUN_ARGS='--src . --steps 8 "Write a README.md and then list the directory"'
- Require a tool: make run RUN_ARGS='--tool-choice required --require-tool write_file "Write then read a file"'

## Environment

- OPENAI_API_KEY must be set for real API calls
- Logging can be toggled with `--log` flag (enabled by default)

## CI / Pre-commit (optional suggestion)

Locally, you can simulate a pre-commit flow:
- make fmt
- make lint
- make test

## Troubleshooting

- If gum/staticcheck/goimports/vhs are missing, run `make tools`
- If `make tools` can’t install, install Go from https://go.dev/dl/ and re-run
- On Windows shells without make, use the raw Go commands above

## Questions

Open an issue or start a discussion. Contributions are welcome!