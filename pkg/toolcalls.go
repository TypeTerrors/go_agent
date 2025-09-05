package pkg

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/openai/openai-go/v2"
)

// ToolCallLite is a compact, SDK-agnostic tool call used for planning.
// Flow: created after model returns tool calls, before planning.
type ToolCallLite struct {
	ID       string
	FuncName string
	FuncArgs string
	PathAbs  string // absolute file path for file ops
	DirAbs   string // absolute directory for list_dir
}

// AnalyzeLite extracts absolute path and directory for a tool call.
// Flow: used by PlanPhases() to populate fields for ordering.
func AnalyzeLite(root, name, rawArgs string) (absPath, dir string) {
	var a map[string]any
	_ = json.Unmarshal([]byte(rawArgs), &a)
	switch name {
	case "read_file", "write_file", "delete_path":
		p := filepath.FromSlash(fmt.Sprint(a["path"]))
		if p != "" {
			abs := filepath.Join(root, p)
			clean := filepath.Clean(abs)
			return clean, filepath.Dir(clean)
		}
	case "list_dir":
		d := filepath.FromSlash(fmt.Sprint(a["dir"]))
		if d == "" {
			d = "."
		}
		abs := filepath.Join(root, d)
		return "", filepath.Clean(abs)
	case "run_command":
		return "", root
	}
	return "", ""
}

// ExtractToolCalls converts SDK tool calls into ToolCallLite slice.
// Flow: called in Run() immediately after receiving assistant message.
func ExtractToolCalls(msg openai.ChatCompletionMessage) []ToolCallLite {
	out := make([]ToolCallLite, len(msg.ToolCalls))
	for i, tc := range msg.ToolCalls {
		out[i] = ToolCallLite{
			ID:       tc.ID,
			FuncName: tc.Function.Name,
			FuncArgs: tc.Function.Arguments,
		}
	}
	return out
}
