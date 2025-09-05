package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"cds.agents.app/internal/services/agent"
)

// makeNested is a helper to create a temporary project root with nested files.
// It returns the root directory path.
func makeNested(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	must := func(err error) { if err != nil { t.Fatalf("%v", err) } }

	must(os.MkdirAll(filepath.Join(root, "a/b/c"), 0o755))
	must(os.WriteFile(filepath.Join(root, "a", "x.txt"), []byte("x"), 0o644))
	must(os.WriteFile(filepath.Join(root, "a", "b", "y.txt"), []byte("y"), 0o644))
	must(os.WriteFile(filepath.Join(root, "a", "b", "c", "z.txt"), []byte("z"), 0o644))
	return root
}

// newTestAgent creates a new test agent with the specified root directory.
// It returns the initialized agent.
func newTestAgent(root string) *agent.Agent {
	a := &agent.Agent{}
	a.Init("gpt-4o", root, 2, 1, 0, "test")
	return a
}

// TestListDirRecursive tests the list_dir_recursive tool.
func TestListDirRecursive(t *testing.T) {
	root := makeNested(t)
	a := newTestAgent(root)
	out, err := a.Tooling(root, "list_dir_recursive", `{"dir":"a"}`)
	if err != nil { t.Fatalf("list_dir_recursive err: %v", err) }
	// Expect to see nested entries
	if !strings.Contains(out, "DIR  b") { t.Fatalf("expected DIR  b in output, got:\n%s", out) }
	if !strings.Contains(out, "FILE b/y.txt") { t.Fatalf("expected FILE b/y.txt in output, got:\n%s", out) }
	if !strings.Contains(out, "FILE b/c/z.txt") { t.Fatalf("expected FILE b/c/z.txt in output, got:\n%s", out) }
}

// TestWriteReadFile tests the write_file and read_file tools.
func TestWriteReadFile(t *testing.T) {
	root := t.TempDir()
	a := newTestAgent(root)
	_, err := a.Tooling(root, "write_file", `{"path":"foo/bar.txt","content":"hello"}`)
	if err != nil { t.Fatalf("write_file err: %v", err) }
	out, err := a.Tooling(root, "read_file", `{"path":"foo/bar.txt"}`)
	if err != nil { t.Fatalf("read_file err: %v", err) }
	if out != "hello" { t.Fatalf("unexpected content: %q", out) }
}

// TestDeletePathRecursive tests the delete_path tool.
func TestDeletePathRecursive(t *testing.T) {
	root := makeNested(t)
	a := newTestAgent(root)
	// ensure something exists
	if _, err := os.Stat(filepath.Join(root, "a", "b")); err != nil { t.Fatalf("precheck: %v", err) }
	_, err := a.Tooling(root, "delete_path", `{"path":"a/b"}`)
	if err != nil { t.Fatalf("delete_path err: %v", err) }
	if _, err := os.Stat(filepath.Join(root, "a", "b")); !os.IsNotExist(err) {
		t.Fatalf("expected a/b to be gone, err=%v", err)
	}
}

// TestRunCommand_Success tests a simple echo command.
func TestRunCommand_Success(t *testing.T) {
	root := t.TempDir()
	a := newTestAgent(root)
	out, err := a.Tooling(root, "run_command", `{"cmd":"echo hello","permissions":"r"}`)
	if err != nil { t.Fatalf("run_command err: %v", err) }
	if !strings.Contains(out, "hello") { t.Fatalf("expected echo output, got: %q", out) }
}

// TestRunCommand_WriteDenied ensures write-like ops need 'w'.
func TestRunCommand_WriteDenied(t *testing.T) {
	root := t.TempDir()
	a := newTestAgent(root)
	_, err := a.Tooling(root, "run_command", `{"cmd":"sh -lc 'echo hi > f.txt'","permissions":"r"}`)
	if err == nil { t.Fatalf("expected error for write without 'w'") }
}

// TestRunCommand_ExecRequiresX ensures path exec needs 'x'.
func TestRunCommand_ExecRequiresX(t *testing.T) {
	if runtime.GOOS == "windows" { t.Skip("bash dependency not available on windows in CI") }
	root := t.TempDir()
	a := newTestAgent(root)
	// create a small script
	script := filepath.Join(root, "tool.sh")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env bash\necho ok\n"), 0o755); err != nil { t.Fatalf("write script: %v", err) }
	_, err := a.Tooling(root, "run_command", `{"cmd":"./tool.sh","permissions":"r"}`)
	if err == nil { t.Fatalf("expected error for exec without 'x'") }
	out, err := a.Tooling(root, "run_command", `{"cmd":"./tool.sh","permissions":"rx"}`)
	if err != nil { t.Fatalf("unexpected err with x: %v", err) }
	if !strings.Contains(out, "ok") { t.Fatalf("expected ok, got: %q", out) }
}

// TestRunCommand_Timeout validates timeout behavior.
func TestRunCommand_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" { t.Skip("bash dependency not available on windows in CI") }
	root := t.TempDir()
	a := newTestAgent(root)
	start := time.Now()
	_, err := a.Tooling(root, "run_command", `{"cmd":"sleep 1","permissions":"r","timeout":"200ms"}`)
	if err == nil { t.Fatalf("expected timeout error") }
	if time.Since(start) > 2*time.Second { t.Fatalf("timeout did not trigger promptly") }
}
