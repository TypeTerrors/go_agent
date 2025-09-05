package main

import (
	"fmt"
	"os"

	"cds.agents.app/internal/cli"
	"github.com/charmbracelet/fang"
)

// main wires the CLI and starts command execution.
// Flow: entrypoint of the binary.
// Yields: process exit after command completes.
func main() {
	root := cli.BuildRootCmd()
	if err := cli.Execute(root, fang.WithoutCompletions(), fang.WithoutManpage(), fang.WithoutVersion()); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
