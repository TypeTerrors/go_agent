package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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

// NewAgent constructs the Agent with initial configuration.
// Flow: called by CLI to create the agent before any execution.
// Yields: no yielding; prepares runtime state.
func NewAgent(config pkg.Config) *Agent {
	agent := &Agent{}
	agent.Init(config.Model, config.Src, config.Concurrency, config.Steps, config.Timeout, config.Prompt)
	agent.ToolChoice = config.ToolChoice
	agent.RequireTools = config.RequireTools
	lg := pkg.NewLogger(config.Log)
	agent.Log = lg
	return agent
}

// Init sets client, locks, and runtime parameters.
// Flow: invoked by NewAgent prior to running.
// Yields: no yielding; configuration only.
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

// Run is the main loop: prompt -> model -> tools -> results -> repeat.
// Flow: top-level execution after construction.
// Yields: returns when assistant has no tool calls or when steps exhausted.
func (a *Agent) Run() error {

	// construct initial prompt
	a.Prompt()

	a.printConfig()

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

// printConfig logs startup settings for visibility.
// Flow: called once after Prompt() in Run().
// Yields: no yielding; side-effect logging.
func (a *Agent) printConfig() {
	a.Log.Info("")
	a.Log.Info("  Using model: " + a.Model)
	a.Log.Info("  Current src: " + a.Src)
	a.Log.Info(fmt.Sprintf("  Max steps  : %d", a.Steps))
	a.Log.Info(fmt.Sprintf("  Timeout    : %s", a.Timeout.String()))
	a.Log.Info(fmt.Sprintf("  Concurrency: %d", a.Concurrency))
	if a.ToolChoice != "" {
		a.Log.Info("  Tool choice: " + a.ToolChoice)
	}
	if len(a.RequireTools) > 0 {
		a.Log.Info("  Need Tools : " + strings.Join(a.RequireTools, ", "))
	}
	a.Log.Info("")
}

// setClient establishes the OpenAI client with tool choice.
// Flow: during Init.
// Yields: none.
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

// setModel stores the LLM model identifier.
// Flow: during Init.
// Yields: none.
func (a *Agent) setModel(model string) {
	a.Model = model
}

// setSrc sets the working source directory.
// Flow: during Init.
// Yields: none.
func (a *Agent) setSrc(src string) {
	a.Src = src
}

// setConcurrency limits parallelism per phase.
// Flow: during Init.
// Yields: none.
func (a *Agent) setConcurrency(concurrency int) {
	a.Concurrency = concurrency
}

// setSteps defines the maximum assistant turns.
// Flow: during Init.
// Yields: none.
func (a *Agent) setSteps(steps int) {
	a.Steps = steps
}

// setTimeout defines per-turn API timeout.
// Flow: during Init.
// Yields: none.
func (a *Agent) setTimeout(timeout time.Duration) {
	a.Timeout = timeout
}

// setLockManager prepares per-path locks for FS tools.
// Flow: during Init.
// Yields: none.
func (a *Agent) setLockManager() {
	a.Lm = pkg.NewLockManager()
}

// setPrompt records the initial natural-language task.
// Flow: during Init.
// Yields: none.
func (a *Agent) setPrompt(q string) {
	a.Query = q
}
