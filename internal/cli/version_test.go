package cli

import (
	"bytes"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/JanDeDobbeleer/copilot-ralph/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunVersionShort(t *testing.T) {
	old := version.Version
	defer func() { version.Version = old }()
	version.Version = "9.9.9"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	versionShort = true

	runVersion(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := strings.TrimSpace(buf.String())

	require.Equal(t, "9.9.9", out)
}

func TestRunVersionFull(t *testing.T) {
	oldV, oldC, oldD, oldG := version.Version, version.Commit, version.BuildDate, version.GoVersion
	defer func() { version.Version, version.Commit, version.BuildDate, version.GoVersion = oldV, oldC, oldD, oldG }()
	version.Version = "1.2.3"
	version.Commit = "abc"
	version.BuildDate = "2026-01-01"
	version.GoVersion = "go1.20"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	versionShort = false

	runVersion(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	assert.Contains(t, out, "Ralph v1.2.3")
	assert.Contains(t, out, "Commit: abc")
	assert.Contains(t, out, "Built: 2026-01-01")
	assert.Contains(t, out, "Go: go1.20")
	assert.Contains(t, out, "Platform: "+runtime.GOOS+"/"+runtime.GOARCH)
}
