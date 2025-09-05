package pkg

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/openai/openai-go/v2"
)

// ToolCallLite provides a local, SDK-agnostic view of a tool call.
// It contains only the necessary information for planning.
type ToolCallLite struct {
	ID       string
	FuncName string
	FuncArgs string
	PathAbs  string // absolute file path for file ops
	DirAbs   string // absolute directory for list_dir
}

// AnalyzeLite extracts path/dir from args for dependency building.
// Parameters:
// - root: the root directory for the operations.
// - name: the name of the tool.
// - rawArgs: the raw JSON string of arguments for the tool.
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
	}
	return "", ""
}

// ExtractToolCalls converts SDK tool calls to a slice of ToolCallLite.
// Parameters:
// - msg: the OpenAI chat completion message containing the tool calls.
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
