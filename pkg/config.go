package pkg

import "time"

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
