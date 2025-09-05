package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
