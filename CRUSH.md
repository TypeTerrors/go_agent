CRUSH.md — Commands and Conventions for this repo

Build / Run / Test
- Build all: go build ./...
- Build CLI: go build -o bin/agent ./cmd/agent
- Run CLI: go run ./cmd/agent --help
- Run tests (all): go test ./...
- Run a single test file: go test ./tests -run TestName
- Run a single test (regex): go test ./tests -run '^TestWriteReadFile$'
- Race + verbose: go test -race -v ./...
- Lint (suggested): go vet ./...; staticcheck ./... (install staticcheck first)
- Tidy modules: go mod tidy

Project layout
- cmd/agent: entrypoint main; internal/services/agent: core agent logic; internal/services/prompts: prompt text; pkg/: shared utilities.

Code style
- Imports: standard lib, then third‑party, then local (cds.agents.app/...). Keep groups separated with blank lines. Use goimports or gofmt.
- Formatting: gofmt/goimports enforced; no custom formatting.
- Types: prefer concrete types; expose minimal surface area. Return (val, error) rather than panicking.
- Naming: MixedCaps for exported, mixedCaps for unexported. Keep package names short, lower case, no underscores.
- Errors: use error values; wrap with fmt.Errorf("context: %w", err). Do not ignore errors. Prefer sentinel errors or errors.Is/As.
- Logging: use pkg/logger where available; avoid fmt.Println in library code; keep logs structured and concise.
- Concurrency: use context when adding external I/O; prefer x/sync/errgroup for groups. Protect shared state.
- Filesystem helpers: use pkg/toolcalls and internal/services/agent tools consistently.
- Testing: table-driven where practical; use t.Helper(); avoid time.Sleep; assert via stdlib testing only.

Security and secrets
- Never log API keys; read from env/config only. Do not commit secrets. Validate user paths; avoid directory traversal issues.

Cursor/Copilot rules
- No Cursor rules or Copilot instruction files detected; follow the above conventions.
