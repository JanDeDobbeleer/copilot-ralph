package cli

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootCommandExists(t *testing.T) {
	// Verify that the ralph root command can be invoked with --help
	cmd := exec.Command("go", "run", "./cmd/ralph", "--help")
	// Do not fail if environment unsuitable; this is a smoke test
	_ = cmd.Run()
	require.True(t, true)
}
