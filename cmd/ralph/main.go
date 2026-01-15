// Package main is the entry point for the Ralph CLI application.
//
// Ralph implements the "Ralph Wiggum" technique for iterative AI development loops.
// It continuously feeds prompts to GitHub Copilot, monitoring for completion signals.
//
// See specs/ directory for detailed specifications.
package main

import (
	"os"

	"github.com/JanDeDobbeleer/copilot-ralph/internal/cli"
)

func main() {
	// TODO: Implement main per specs/cli.md
	// Initialize root command and execute
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
