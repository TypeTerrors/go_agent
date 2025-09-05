Safely run a shell command inside the source sandbox.

Inputs
- cmd: string — the full command line to execute
- permissions: string — subset of rwx characters controlling capabilities
  - r: allow read-only commands (e.g., cat, git status, go env)
  - w: allow write/mutate operations (e.g., git add/commit, go mod tidy)
  - x: allow execution of binaries within the sandbox
- timeout: string — optional duration (e.g., "60s"); defaults to 60s

Rules
- Working directory is pinned to the project source; paths must not escape the sandbox.
- Outputs are captured (stdout/stderr) and may be truncated.
- Dangerous commands (rm -rf /, sudo, mount, networking tools) are blocked regardless of permissions.
- Prefer minimal permissions; only request what you need.

Return
- Combined stdout/stderr text and an exit status note.