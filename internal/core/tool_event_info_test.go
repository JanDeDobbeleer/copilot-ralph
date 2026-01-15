package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToolEventInfo(t *testing.T) {
	// No parameters
	e := &ToolEvent{ToolName: "echo", Parameters: map[string]interface{}{}, Iteration: 1}
	info := e.Info("!")
	assert.Equal(t, "! echo", info)

	// With parameters - values should be present in the returned info
	params := map[string]interface{}{"path": "file.txt", "line": 42}
	e2 := &ToolEvent{ToolName: "edit", Parameters: params, Iteration: 2}
	info2 := e2.Info("🔧")
	assert.Contains(t, info2, "edit")
	assert.Contains(t, info2, "file.txt")
	assert.Contains(t, info2, "42")
}
