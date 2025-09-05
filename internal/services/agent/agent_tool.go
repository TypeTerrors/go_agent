package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)


// Tooling runs a single tool call and returns its textual result.
// Flow: called within RunPhases() concurrently per phase item.
// Yields: returns tool output to be appended as ToolMessage.
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

	case "run_command":
		cmdline := fmt.Sprint(args["cmd"])
		perms := fmt.Sprint(args["permissions"])
		to := fmt.Sprint(args["timeout"]) // e.g., "60s"
		if cmdline == "" {
			return "", errors.New("cmd required")
		}
		le := a.Log.Start("run_command", cmdline)
		// permissions parsing
		_ = strings.Contains(perms, "r") // r currently does not gate execution; kept for future read-only policies
		allowW := strings.Contains(perms, "w")
		allowX := strings.Contains(perms, "x")

		// basic denylist regardless of perms
		deny := []string{"sudo", "mount", "umount", "iptables", "ifconfig", "ssh", "scp", "curl", "wget", "nc", "rm -rf /"}
		for _, d := range deny {
			if strings.Contains(cmdline, d) {
				return "", fmt.Errorf("command contains disallowed token: %s", d)
			}
		}

		// classify mutation attempts
		mutating := false
		writeTokens := []string{"rm ", "mv ", "cp ", "chmod ", "chown ", "git commit", "git add", "git reset", "git revert", "go mod tidy", "sed -i", "tee ", ">", ">>"}
		for _, t := range writeTokens {
			if strings.Contains(cmdline, t) {
				mutating = true
				break
			}
		}
		if mutating && !allowW {
			return "", errors.New("write permissions required (use permissions contains 'w')")
		}

		// disallow executing arbitrary binaries without x
		execBinary := false
		reWord := regexp.MustCompile(`^\s*([a-zA-Z0-9_./-]+)`) // first token
		m := reWord.FindStringSubmatch(cmdline)
		if len(m) > 1 {
			bin := m[1]
			if strings.Contains(bin, "/") {
				execBinary = true
			}
		}
		if execBinary && !allowX {
			return "", errors.New("execute permissions required (include 'x') for running binaries by path")
		}

		// prepare context with timeout
		dur := 60 * time.Second
		if to != "" {
			if parsed, err := time.ParseDuration(to); err == nil && parsed > 0 && parsed <= 5*time.Minute {
				dur = parsed
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), dur)
		defer cancel()

		// execute via shell
		c := exec.CommandContext(ctx, "bash", "-lc", cmdline)
		c.Dir = a.Src
		// inherit limited env, redact secrets from logs; env remains same
		c.Env = os.Environ()
		out, err := c.CombinedOutput()
		text := string(out)
		if len(text) > 4000 {
			text = text[:4000] + "\n...[truncated]"
		}
		if ctx.Err() == context.DeadlineExceeded {
			le.Error(errors.New("timeout"))
			return text + "\n(timeout)", errors.New("command timed out")
		}
		if err != nil {
			le.Error(err)
			// return both output and error for visibility
			return text, err
		}
		le.Success("ok")
		return text, nil

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}
