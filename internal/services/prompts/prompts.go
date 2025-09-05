package prompts

import _ "embed"

// SystemMessage is the embedded system message for the agent.
//go:embed system_message.md
var SystemMessage string

// ListDir is the embedded prompt for the list_dir tool.
//go:embed list_dir.md
var ListDir string

// ListDirRecursive is the embedded prompt for the list_dir_recursive tool.
//go:embed list_dir_recursive.md
var ListDirRecursive string

// ReadFile is the embedded prompt for the read_file tool.
//go:embed read_file.md
var ReadFile string

// WriteFile is the embedded prompt for the write_file tool.
//go:embed write_file.md
var WriteFile string

// DeletePath is the embedded prompt for the delete_path tool.
//go:embed delete_path.md
var DeletePath string
