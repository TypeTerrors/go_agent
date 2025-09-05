package pkg

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go/v2"
)

// Config holds the configuration for the agent.
type Config struct {
	Model        string
	Src          string
	Concurrency  int
	Steps        int
	Timeout      time.Duration
	Prompt       string
	Log          bool
	ToolChoice   string
	RequireTools []string
}

// ParseFlags parses command-line flags and returns a Config.
// It sets default values and validates the prompt.
func ParseFlags() Config {
	src := flag.String("src", ".", "source directory to operate in (defaults to current directory)")
	concurrency := flag.Int("concurrency", 4, "max concurrent tool executions per phase")
	steps := flag.Int("steps", 16, "max assistant turns (avoid infinite loops)")
	model := flag.String("model", string(openai.ChatModelGPT4o), "OpenAI chat model (e.g., gpt-4o)")
	timeout := flag.Duration("timeout", 120*time.Second, "per-turn API timeout")
	logEnabled := flag.Bool("log", true, "enable pretty CLI logs")
	toolChoice := flag.String("tool-choice", "auto", "tool choice behavior: auto|required|none")
	var requireTools multiString
	flag.Var(&requireTools, "require-tool", "require a specific tool to be used (repeatable)")
	flag.Parse()

	prompt := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if prompt == "" {
		fmt.Println("usage: go run ./cmd/agent --src . --concurrency 6 --tool-choice required --require-tool write_file \"Create README.md and list the directory.\"")
		os.Exit(2)
	}

	return Config{
		Model:        *model,
		Src:          *src,
		Concurrency:  *concurrency,
		Steps:        *steps,
		Timeout:      *timeout,
		Prompt:       prompt,
		Log:          *logEnabled,
		ToolChoice:   *toolChoice,
		RequireTools: requireTools,
	}
}

type multiString []string

func (m *multiString) String() string { return strings.Join(*m, ",") }
func (m *multiString) Set(s string) error {
	*m = append(*m, s)
	return nil
}
