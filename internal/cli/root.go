// Package cli implements the command-line interface for Ralph using Cobra.
//
// This package defines all CLI commands (run, init, version) and their flags.
// It orchestrates the execution flow between TUI and Core components.
//
// See specs/cli.md for detailed CLI specification.
package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/JanDeDobbeleer/copilot-ralph/pkg/version"
)

var (
	// noColor disables colored output
	noColor bool

	// rootCmd is the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "ralph",
		Short: "Ralph - Iterative AI Development Loop Tool",
		Long: `Ralph implements the "Ralph Wiggum" technique for self-referential AI
development loops using GitHub Copilot and Bubble Tea TUI.`,
		Version: version.Version,
	}
)

// Execute runs the root command and returns any error.
func Execute(ctx context.Context) error {
	if ctx == nil {
		return rootCmd.Execute()
	}

	return rootCmd.ExecuteContext(ctx)
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")

	// Add subcommands
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
}
