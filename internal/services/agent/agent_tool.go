package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Tooling executes a tool call based on the provided name and arguments.
// It ensures operations are confined within the source directory.
// Parameters:
// - root: the root directory for the operations.
// - name: the name of the tool to be executed.
// - rawArgs: the raw JSON string of arguments for the tool.
func (a *Agent) Tooling(root string, name string, rawArgs string) (string, error) {
	// Parse JSON args
	var args map[string]any
	_ = json.Unmarshal([]byte(rawArgs), &args)

	// Path guard: stay inside src directory
	resolve := func(rel string) (string, error) {
		if rel == "" {
			return "", errors.New("path required")
		}
		abs := filepath.Join(root, filepath.FromSlash(rel))
		relBack, err := filepath.Rel(root, abs)
		if err != nil || strings.HasPrefix(relBack, "..") {
			return "", errors.New("refusing to access outside source directory")
		}
		return filepath.Clean(abs), nil
	}

	switch name {
	case "list_dir":
		dir := fmt.Sprint(args["dir"])
		le := a.Log.Start("list_dir", dir)
		if dir == "" {
			dir = "."
		}
		abs, err := resolve(dir)
		if err != nil {
			return "", err
		}
		mu := a.Lm.Get(abs)
		mu.RLock()
		defer mu.RUnlock()

		ents, err := os.ReadDir(abs)
		if err != nil {
			le.Error(err)
			return "", err
		}
		var b strings.Builder
		for _, e := range ents {
			if e.IsDir() {
				b.WriteString("DIR  " + e.Name() + "\n")
			} else {
				b.WriteString("FILE " + e.Name() + "\n")
			}
		}
		le.Success(fmt.Sprintf("%d entries", len(ents)))
		return b.String(), nil

	case "list_dir_recursive":
		dir := fmt.Sprint(args["dir"])
		le := a.Log.Start("list_dir_recursive", dir)
		if dir == "" {
			dir = "."
		}
		abs, err := resolve(dir)
		if err != nil {
			return "", err
		}
		mu := a.Lm.Get(abs)
		mu.RLock()
		defer mu.RUnlock()

		var out []string
		err = filepath.WalkDir(abs, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(abs, p)
			if rel == "." {
				return nil
			}
			if d.IsDir() {
				out = append(out, "DIR  "+rel)
			} else {
				out = append(out, "FILE "+rel)
			}
			return nil
		})
		if err != nil {
			le.Error(err)
			return "", err
		}
		le.Success(fmt.Sprintf("%d entries", len(out)))
		return strings.Join(out, "\n"), nil

	case "read_file":
		p := fmt.Sprint(args["path"])
		le := a.Log.Start("read_file", p)
		abs, err := resolve(p)
		if err != nil {
			return "", err
		}
		mu := a.Lm.Get(abs)
		mu.RLock()
		defer mu.RUnlock()

		b, err := os.ReadFile(abs)
		if err != nil {
			le.Error(err)
			return "", err
		}
		le.Success(fmt.Sprintf("%d bytes", len(b)))
		return string(b), nil

	case "write_file":
		p := fmt.Sprint(args["path"])
		le := a.Log.Start("write_file", p)
		content := fmt.Sprint(args["content"])
		abs, err := resolve(p)
		if err != nil {
			return "", err
		}
		mu := a.Lm.Get(abs)
		mu.Lock()
		defer mu.Unlock()

		if err := a.Lm.WriteAtomic(abs, []byte(content)); err != nil {
			le.Error(err)
			return "", err
		}
		le.Success(fmt.Sprintf("%d bytes", len(content)))
		return fmt.Sprintf("wrote %s (%d bytes)", p, len(content)), nil

	case "delete_path":
		p := fmt.Sprint(args["path"])
		le := a.Log.Start("delete_path", p)
		abs, err := resolve(p)
		if err != nil {
			return "", err
		}
		mu := a.Lm.Get(abs)
		mu.Lock()
		defer mu.Unlock()

		if err := os.RemoveAll(abs); err != nil {
			le.Error(err)
			return "", err
		}
		le.Success("deleted")
		return "deleted " + p, nil

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}
