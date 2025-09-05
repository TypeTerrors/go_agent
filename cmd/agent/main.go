package main

import (
	"fmt"
	"os"

	"cds.agents.app/internal/cli"
	"github.com/charmbracelet/fang"
)

func main() {
	root := cli.BuildRootCmd()
	if err := cli.Execute(root, fang.WithoutCompletions(), fang.WithoutManpage(), fang.WithoutVersion()); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
