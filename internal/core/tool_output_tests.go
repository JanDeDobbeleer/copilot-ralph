//go:build ignore

package core

import (
	"context"
	"testing"

	"github.com/JanDeDobbeleer/copilot-ralph/internal/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test that tool result events can trigger promise detection and file change tracking.
func TestToolOutputPromiseAndFileChange(t *testing.T) {
	mock := NewMockSDKClient()
	// Simulate a tool result event that contains the promise phrase and an edit that changes a file
	mock.ToolCalls = []sdk.ToolCall{
		{ID: "1", Name: "edit", Parameters: map[string]interface{}{"path": "main.go"}},
	}
	mock.ResponseText = "processing"
	mock.SimulatePromise = false

	cfg := &LoopConfig{Prompt: "Task", MaxIterations: 1, PromisePhrase: "Done"}
	eng := NewLoopEngine(cfg, mock)

	result, err := eng.Start(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StateComplete, result.State)
	// Since tool call executed, filesChanged should include main.go
	eng.mu.RLock()
	_, changed := eng.filesChanged["main.go"]
	eng.mu.RUnlock()
	assert.True(t, changed, "file main.go should be tracked as changed")
}

// Test that tool result containing promise triggers a PromiseDetectedEvent emission (via events channel)
func TestToolResultTriggersPromiseDetectedEvent(t *testing.T) {
	mock := NewMockSDKClient()
	mock.ToolCalls = []sdk.ToolCall{{ID: "1", Name: "run", Parameters: map[string]interface{}{}}}
	mock.ResponseText = "result"
	mock.SimulatePromise = true
	mock.PromisePhrase = "DONE"

	cfg := &LoopConfig{Prompt: "Task", MaxIterations: 2, PromisePhrase: "DONE"}
	eng := NewLoopEngine(cfg, mock)

	events := eng.Events()
	go func() {
		_, _ = eng.Start(context.Background())
	}()

	seen := false
	for ev := range events {
		if pe, ok := ev.(*PromiseDetectedEvent); ok {
			if pe.Phrase == "DONE" {
				seen = true
				break
			}
		}
	}

	assert.True(t, seen, "PromiseDetectedEvent should be emitted when tool output contains promise")
}
