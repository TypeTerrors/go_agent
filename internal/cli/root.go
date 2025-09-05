package cli

import (
	"context"
	"strings"
	"time"

	"cds.agents.app/internal/services/agent"
	"cds.agents.app/pkg"
	"github.com/charmbracelet/fang"
	"github.com/openai/openai-go/v2"
	"github.com/spf13/cobra"
)

// BuildRootCmd defines the CLI and binds flags to config.
// Flow: called by main() to construct the root command.
// Yields: no; returns cobra.Command to execute.
func BuildRootCmd() *cobra.Command {
	var (
		src         string
		concurrency int
		steps       int
		model       string
		timeout     time.Duration
		logEnabled   bool
		toolChoice   string
		requireTools []string

	)

	root := &cobra.Command{
		Use:   "agent [flags] \"task prompt\"",
		Short: "Iterative tool-calling code mod agent",
		Long:  "Agent CLI — plans and executes filesystem tools iteratively to accomplish coding tasks.\n\nExamples:\n  agent --src . --concurrency 6 --steps 16 \"Create README.md and list the directory.\"\n  agent --tool-choice required --require-tool write_file \"Write 'hello' to README.md and then read it.\"\n  agent --tool-choice none \"Explain what this tool does.\"\n  agent --log=true --steps=1000 \"make two short stories in seperate .md files\"",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			prompt := strings.TrimSpace(strings.Join(args, " "))
			config := pkg.Config{
				Model:        model,
				Src:          src,
				Concurrency:  concurrency,
				Steps:        steps,
				Timeout:      timeout,
				Prompt:       prompt,
				Log:          logEnabled,
				ToolChoice:   toolChoice,
				RequireTools: requireTools,

			}
			a := agent.NewAgent(config)
			return a.Run()
		},
	}

	root.Flags().StringVar(&src, "src", ".", "source directory to operate in (defaults to current directory)")
	root.Flags().IntVar(&concurrency, "concurrency", 4, "max concurrent tool executions per phase")
	root.Flags().IntVar(&steps, "steps", 16, "max assistant turns (avoid infinite loops)")
	root.Flags().StringVar(&model, "model", string(openai.ChatModelGPT4o), "OpenAI chat model (e.g., gpt-4o)")
	root.Flags().DurationVar(&timeout, "timeout", 600*time.Second, "per-turn API timeout")
	root.Flags().BoolVar(&logEnabled, "log", true, "enable pretty CLI logs")

	root.Flags().StringVar(&toolChoice, "tool-choice", "auto", "tool choice behavior: auto|required|none")
	root.Flags().StringArrayVar(&requireTools, "require-tool", nil, "require a specific tool to be used (repeatable)")

	return root
}

// Execute runs the CLI command with Fang integration.
// Flow: called by main() to start command handling.
// Yields: returns error for process exit handling.
func Execute(root *cobra.Command, opts ...fang.Option) error {
	return fang.Execute(context.Background(), root, opts...)
}
