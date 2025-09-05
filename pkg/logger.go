package pkg

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Logger provides logging capabilities for the agent.
type Logger struct{
	enabled bool
}

// NewLogger creates a new Logger instance.
// Parameters:
// - enabled: a boolean indicating if logging is enabled.
func NewLogger(enabled bool) *Logger { return &Logger{enabled: enabled} }

// LogEntry represents a single log entry.
type LogEntry struct{
	l *Logger
	tool string
	target string
	start time.Time
}

var (
	styleTool      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	stylePath      = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	styleInfo      = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleError     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleOK        = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	styleAsstTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))
	styleAsstBody  = lipgloss.NewStyle().Foreground(lipgloss.Color("219"))
)

func StyleAsstTitle() lipgloss.Style { return styleAsstTitle }
func StyleAsstBody() lipgloss.Style  { return styleAsstBody }
func StyleInfo() lipgloss.Style      { return styleInfo }
func StylePath() lipgloss.Style      { return stylePath }
func StyleTool() lipgloss.Style      { return styleTool }
func StyleOK() lipgloss.Style        { return styleOK }
func StyleError() lipgloss.Style     { return styleError }

// Start begins a new log entry for a tool operation.
// Parameters:
// - tool: the name of the tool being executed.
// - target: the target of the tool operation.
func (l *Logger) Start(tool, target string) *LogEntry {
	if l == nil || !l.enabled { return &LogEntry{} }
	le := &LogEntry{l: l, tool: tool, target: target, start: time.Now()}
	fmt.Fprintf(os.Stderr, "%s %s %s\n",
		styleInfo.Render(""),
		styleTool.Render(tool),
		stylePath.Render(target),
	)
	return le
}

// Success logs a successful tool operation.
// Parameters:
// - msg: a message describing the success.
func (le *LogEntry) Success(msg string) {
	if le == nil || le.l == nil || !le.l.enabled { return }
	d := time.Since(le.start)
	fmt.Fprintf(os.Stderr, "%s %s (%s) %s\n",
		styleOK.Render(""),
		styleTool.Render(le.tool),
		styleInfo.Render(d.Truncate(time.Millisecond).String()),
		msg,
	)
}

// Error logs an error that occurred during a tool operation.
// Parameters:
// - err: the error that occurred.
func (le *LogEntry) Error(err error) {
	if le == nil || le.l == nil || !le.l.enabled { return }
	d := time.Since(le.start)
	fmt.Fprintf(os.Stderr, "%s %s (%s) %s\n",
		styleError.Render(""),
		styleTool.Render(le.tool),
		styleInfo.Render(d.Truncate(time.Millisecond).String()),
		styleError.Render(err.Error()),
	)
}

// Info logs a simple informational line.
func (l *Logger) Info(msg string) {
	if l == nil || !l.enabled { return }
	fmt.Fprintf(os.Stderr, "%s %s\n", styleInfo.Render("ℹ"), styleInfo.Render(msg))
}

// PrintAssistant logs the assistant's response.
// Parameters:
// - content: the content of the assistant's response.
func (l *Logger) PrintAssistant(content string) {
	if l == nil || !l.enabled {
		fmt.Println("\n--- ASSISTANT ---\n" + content)
		return
	}
	fmt.Fprintln(os.Stderr, styleAsstTitle.Render("\n--- ASSISTANT ---"))
	fmt.Fprintln(os.Stderr, styleAsstBody.Render(content))
}
