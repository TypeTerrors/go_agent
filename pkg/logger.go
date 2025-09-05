package pkg

import (
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/lipgloss"
)

type Logger struct{
	enabled bool
	l *log.Logger
	devCaller bool
}


func NewLogger(enabled bool) *Logger {
	lg := log.New(os.Stderr)
	lg.SetLevel(log.InfoLevel)
	lg.SetReportTimestamp(true)
	lg.SetFormatter(log.TextFormatter)
	// Prefix flair
	lg.SetPrefix("AGENT")
	styles := log.DefaultStyles()
	styles.Timestamp = styles.Timestamp.Faint(true)
	styles.Message = styles.Message.Faint(true).Bold(false)
	styles.Prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1)
	// Level badges with icons
	styles.Levels[log.InfoLevel] = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")).SetString("ℹ")
	styles.Levels[log.WarnLevel] = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).SetString("⚠")
	styles.Levels[log.ErrorLevel] = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).SetString("✖")
	styles.Levels[log.DebugLevel] = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("135")).SetString("•")
	// Neon-like key styling
	styles.Keys["tool"] = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Faint(true)
	styles.Values["tool"] = lipgloss.NewStyle().Bold(true).Faint(true)
	styles.Keys["target"] = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Faint(true)
	styles.Values["target"] = lipgloss.NewStyle().Bold(true).Faint(true)
	styles.Keys["dur"] = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Faint(true)
	styles.Keys["msg"] = lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Bold(true)
	styles.Keys["err"] = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styles.Values["err"] = lipgloss.NewStyle().Bold(true)
	lg.SetStyles(styles)
	dev := os.Getenv("LOG_CALLER") == "1"
	lg.SetReportCaller(dev)
	if dev {
		lg.SetCallerFormatter(log.ShortCallerFormatter)
	}
	return &Logger{enabled: enabled, l: lg, devCaller: dev}
}

type LogEntry struct{
	l *Logger
	tool string
	target string
	start time.Time
	sub *log.Logger
}

func (le *LogEntry) Preview(title, content string) {
	// no-op: previews disabled
}

func (l *Logger) Start(tool, target string) *LogEntry {
	if l == nil || !l.enabled { return &LogEntry{} }
	sub := l.l.With("tool", tool, "target", target)
	sub.Info("start")
	return &LogEntry{l: l, tool: tool, target: target, start: time.Now(), sub: sub}
}

func (le *LogEntry) Success(msg string) {
	if le == nil || le.l == nil || !le.l.enabled { return }
	d := time.Since(le.start).Truncate(time.Millisecond)
	le.sub.Info("ok", "dur", d.String(), "msg", msg)
}

func (le *LogEntry) Error(err error) {
	if le == nil || le.l == nil || !le.l.enabled { return }
	d := time.Since(le.start).Truncate(time.Millisecond)
	le.sub.Error("error", "dur", d.String(), "err", err)
}

func (l *Logger) Info(msg string) {
	if l == nil || !l.enabled { return }
	l.l.Info(msg)
}

func (l *Logger) PrintAssistant(content string) {
	if l == nil {
		return
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213")).Render("\n--- ASSISTANT ---")
	body := lipgloss.NewStyle().Foreground(lipgloss.Color("219")).Render(content)
	if !l.enabled {
		os.Stdout.WriteString(title + "\n" + body + "\n")
		return
	}
	l.l.Info("assistant")
	os.Stderr.WriteString(title + "\n")
	os.Stderr.WriteString(body + "\n")
}
