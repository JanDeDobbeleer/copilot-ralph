// Package cli implements the command-line interface for Ralph using Cobra.
//
// This file implements the `ralph version` command for displaying version information.
//
// See specs/cli.md for detailed CLI specification.
package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/JanDeDobbeleer/copilot-ralph/pkg/version"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: `Display version information for Ralph.

Examples:
  # Full version information
  ralph version

  # Short version only
  ralph version --short`,
	Run: runVersion,
}

var versionShort bool

func init() {
	versionCmd.Flags().BoolVar(&versionShort, "short", false, "show only version number")
}

// runVersion executes the version command.
func runVersion(cmd *cobra.Command, args []string) {
	info := version.Get()

	if versionShort {
		fmt.Println(info.Version)
		return
	}

	// Full version output
	fmt.Printf("Ralph v%s\n", info.Version)
	fmt.Printf("Commit: %s\n", info.Commit)
	fmt.Printf("Built: %s\n", info.BuildDate)
	fmt.Printf("Go: %s\n", info.GoVersion)
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
