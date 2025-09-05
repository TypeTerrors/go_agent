package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
	"strings"

	"cds.agents.app/pkg"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

// Agent represents the main structure for the agent.
// It holds configuration and state for the agent's operation.
type Agent struct {
	Client       openai.Client
	Src          string
	Concurrency  int
	Steps        int
	Model        string
	Timeout      time.Duration
	Params       openai.ChatCompletionNewParams
	Lm           *pkg.LockManager
	Log          *pkg.Logger
	Query        string
	ToolChoice   string
	RequireTools []string
	SettingsView string
}

// NewAgent creates a new Agent instance with the provided configuration.
// It initializes the agent's components.
func NewAgent(config pkg.Config) *Agent {
	agent := &Agent{}
	agent.Init(config.Model, config.Src, config.Concurrency, config.Steps, config.Timeout, config.Prompt)
	agent.ToolChoice = config.ToolChoice
	agent.RequireTools = config.RequireTools
	agent.Log = pkg.NewLogger(config.Log)
	return agent
}

// Init initializes the agent with the given parameters.
// It sets up the client, lock manager, model, source directory, concurrency, steps, timeout, and prompt.
func (a *Agent) Init(model, src string, concurrency, steps int, timeout time.Duration, prompt string) {
	a.setClient()
	a.setLockManager()
	a.setModel(model)
	a.setSrc(src)
	a.setConcurrency(concurrency)
	a.setSteps(steps)
	a.setTimeout(timeout)
	a.setPrompt(prompt)
}

// Run executes the agent's main loop.
// It iteratively interacts with the OpenAI API and processes tool calls.
func (a *Agent) Run() error {

	// construct initial prompt
	a.Prompt()

	a.Log.Info("")
	a.Log.Info("  Using model: " + a.Model)
	a.Log.Info("  Source directory: " + a.Src)
	a.Log.Info(fmt.Sprintf("  Max steps: %d", a.Steps))
	a.Log.Info(fmt.Sprintf("  Timeout per step: %s", a.Timeout.String()))
	a.Log.Info(fmt.Sprintf("  Concurrency: %d", a.Concurrency))
	a.Log.Info("")
	
	if a.ToolChoice != "" {
		a.Log.Info("  Tool choice: " + a.ToolChoice)
	}
	if len(a.RequireTools) > 0 {
		a.Log.Info("  Required tools: " + strings.Join(a.RequireTools, ", "))
	}

	// Turn loop: ask model -> maybe tool calls -> run (phased + parallel) -> feed results -> repeat
	for step := 0; step < a.Steps; step++ {
		ctx, cancel := context.WithTimeout(context.Background(), a.Timeout)
		comp, err := a.Client.Chat.Completions.New(ctx, a.Params)
		cancel()
		if err != nil {
			return fmt.Errorf("openai call: %w", err)
		}
		if len(comp.Choices) == 0 {
			return errors.New("empty completion")
		}

		msg := comp.Choices[0].Message
		a.Params.Messages = append(a.Params.Messages, msg.ToParam()) // record assistant turn (incl. tool calls)

		if len(msg.ToolCalls) == 0 {
			if missing := missingRequiredTools(a.RequireTools, nil); len(missing) > 0 {
				// encourage tool usage next turn
				a.Params.Messages = append(a.Params.Messages, openai.ToolMessage(
					"requirement",
					"The following tools are required but were not called: "+strings.Join(missing, ", ")+". Please call them as needed.",
				))
				continue
			}
			// Done: no tools called — show assistant first, then compact config lines
			a.Log.PrintAssistant(msg.Content)
			return nil
		}

		// Convert SDK tool calls -> local, SDK-agnostic shape.
		toolCalls := pkg.ExtractToolCalls(msg)

		// Build dependency-aware phases for all tool calls in this turn.
		phases, err := a.PlanPhases(a.Src, toolCalls)
		if err != nil {
			return err
		}

		a.RunPhases(toolCalls, phases, msg)

		// After executing tools, verify required tools were called in this turn
		if missing := missingRequiredTools(a.RequireTools, toolCalls); len(missing) > 0 {
			// append reminder so next turn knows
			a.Params.Messages = append(a.Params.Messages, openai.ToolMessage(
				"requirement",
				"Required tools still missing: "+strings.Join(missing, ", ")+". Please call them.",
			))
			continue
		}
	}

	return errors.New("stopped: exceeded max steps")
}

func missingRequiredTools(required []string, calls []pkg.ToolCallLite) []string {
	if len(required) == 0 {
		return nil
	}
	set := map[string]bool{}
	for _, r := range required {
		set[r] = false
	}
	for _, c := range calls {
		if _, ok := set[c.FuncName]; ok {
			set[c.FuncName] = true
		}
	}
	var out []string
	for k, v := range set {
		if !v {
			out = append(out, k)
		}
	}
	return out
}

// setClient initializes the OpenAI client for the agent.
func (a *Agent) setClient() {
	choice := a.ToolChoice
	if choice == "" {
		choice = "auto"
	}
	a.Client = openai.NewClient(
		option.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		option.WithJSONSet("tool_choice", choice),
	)
}

// setModel sets the model for the agent.
func (a *Agent) setModel(model string) {
	a.Model = model
}

// setSrc sets the source directory for the agent.
func (a *Agent) setSrc(src string) {
	a.Src = src
}

// setConcurrency sets the concurrency level for the agent.
func (a *Agent) setConcurrency(concurrency int) {
	a.Concurrency = concurrency
}

// setSteps sets the maximum number of steps for the agent.
func (a *Agent) setSteps(steps int) {
	a.Steps = steps
}

// setTimeout sets the timeout duration for the agent.
func (a *Agent) setTimeout(timeout time.Duration) {
	a.Timeout = timeout
}

// setLockManager initializes the lock manager for the agent.
func (a *Agent) setLockManager() {
	a.Lm = pkg.NewLockManager()
}

// setPrompt sets the initial query prompt for the agent.
func (a *Agent) setPrompt(q string) {
	a.Query = q
}

