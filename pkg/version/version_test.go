package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetReturnsSetValues(t *testing.T) {
	oldV, oldC, oldD, oldG := Version, Commit, BuildDate, GoVersion
	defer func() {
		Version = oldV
		Commit = oldC
		BuildDate = oldD
		GoVersion = oldG
	}()

	Version = "1.2.3"
	Commit = "abc123"
	BuildDate = "2026-01-01"
	GoVersion = "go1.20"

	info := Get()
	assert.Equal(t, "1.2.3", info.Version)
	assert.Equal(t, "abc123", info.Commit)
	assert.Equal(t, "2026-01-01", info.BuildDate)
	assert.Equal(t, "go1.20", info.GoVersion)
}
