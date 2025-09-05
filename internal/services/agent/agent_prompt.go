package agent

import (
	"strings"

	"cds.agents.app/internal/services/prompts"

	openai "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/shared"
)

// Prompt prepares initial messages and tool schemas for the model.
// Flow: called once at the start of Run().
// Yields: none; sets Params for subsequent API call.
func (a *Agent) Prompt() {

	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModel(a.Model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(prompts.SystemMessage),
			openai.UserMessage("Source directory: " + a.Src),
			openai.UserMessage(a.Query),
		},

		// Tools list: keep your existing function tools
		Tools: []openai.ChatCompletionToolUnionParam{
			openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
				Name:        "list_dir",
				Description: openai.String(prompts.ListDir),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]any{
						"dir": map[string]any{"type": "string", "description": "relative directory path"},
					},
					"required": []string{"dir"},
				},
			}),
			openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
				Name:        "list_dir_recursive",
				Description: openai.String(prompts.ListDirRecursive),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]any{
						"dir": map[string]any{"type": "string", "description": "relative directory path"},
					},
					"required": []string{"dir"},
				},
			}),
			openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
				Name:        "read_file",
				Description: openai.String(prompts.ReadFile),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{"type": "string", "description": "relative file path"},
					},
					"required": []string{"path"},
				},
			}),
			openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
				Name:        "write_file",
				Description: openai.String(prompts.WriteFile),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]any{
						"path":    map[string]any{"type": "string"},
						"content": map[string]any{"type": "string"},
					},
					"required": []string{"path", "content"},
				},
			}),
			openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
				Name:        "delete_path",
				Description: openai.String(prompts.DeletePath),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{"type": "string"},
					},
					"required": []string{"path"},
				},
			}),
			openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
				Name:        "run_command",
				Description: openai.String(prompts.RunCommand),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]any{
						"cmd":         map[string]any{"type": "string"},
						"permissions": map[string]any{"type": "string"},
						"timeout":     map[string]any{"type": "string"},
					},
					"required": []string{"cmd"},
				},
			}),
		},
	}

	modelLower := strings.ToLower(a.Model)
	if strings.HasPrefix(modelLower, "gpt-5") || strings.HasPrefix(modelLower, "o") {
		params.ReasoningEffort = shared.ReasoningEffortHigh
	} else {
		params.Temperature = openai.Float(0.1)
	}

	a.Params = params
}
