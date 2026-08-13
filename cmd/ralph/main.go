// Package main is the entry point for the Ralph CLI application.
//
// Ralph implements the "Ralph Wiggum" technique for iterative AI development loops.
// It continuously feeds prompts to GitHub Copilot, monitoring for completion signals.
//
// See specs/ directory for detailed specifications.
package main

import (
	"context"
	"os"

	"github.com/JanDeDobbeleer/copilot-ralph/internal/cli"
)

func main() {
	ctx := context.Background()

	// TODO: Implement main per specs/cli.md
	// Initialize root command and execute
	if err := cli.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
