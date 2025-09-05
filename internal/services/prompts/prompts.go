package prompts

import _ "embed"

// SystemMessage contains the system prompt used at session start.
// Flow: loaded in Prompt() to seed model behavior.
//go:embed system_message.md
var SystemMessage string

// ListDir describes the list_dir tool exposed to the model.
// Flow: registered in Prompt() tool schema.
//go:embed list_dir.md
var ListDir string

// ListDirRecursive describes the list_dir_recursive tool.
// Flow: registered in Prompt() tool schema.
//go:embed list_dir_recursive.md
var ListDirRecursive string

// ReadFile describes the read_file tool.
// Flow: registered in Prompt() tool schema.
//go:embed read_file.md
var ReadFile string

// WriteFile describes the write_file tool.
// Flow: registered in Prompt() tool schema.
//go:embed write_file.md
var WriteFile string

// DeletePath describes the delete_path tool.
// Flow: registered in Prompt() tool schema.
//go:embed delete_path.md
var DeletePath string

// RunCommand describes the run_command tool.
//go:embed run_command.md
var RunCommand string
