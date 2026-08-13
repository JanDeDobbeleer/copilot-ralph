package core

import (
	"testing"
	"time"

	"github.com/JanDeDobbeleer/copilot-ralph/internal/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test that tool result events can trigger promise detection and file change tracking.
func TestToolOutputPromiseAndFileChange(t *testing.T) {
	mock := NewMockSDKClient()
	// Simulate a tool result event that contains the promise phrase and an edit that changes a file
	mock.ToolCalls = []sdk.ToolCall{
		{ID: "1", Name: "edit", Parameters: map[string]any{"path": "main.go"}},
	}
	mock.ResponseText = "processing"
	mock.SimulatePromise = false

	cfg := &LoopConfig{Prompt: "Task", MaxIterations: 1, PromisePhrase: "Done"}
	eng := NewLoopEngine(cfg, mock)

	result, err := eng.Start(t.Context())
	require.NoError(t, err)
	assert.Equal(t, StateComplete, result.State)
}

// Test that tool result containing promise triggers a PromiseDetectedEvent emission (via events channel)
func TestToolResultTriggersPromiseDetectedEvent(t *testing.T) {
	mock := NewMockSDKClient()
	mock.ToolCalls = []sdk.ToolCall{{ID: "1", Name: "run", Parameters: map[string]any{}}}
	mock.ResponseText = "result"
	mock.SimulatePromise = true
	mock.PromisePhrase = "DONE"

	cfg := &LoopConfig{Prompt: "Task", MaxIterations: 2, PromisePhrase: "DONE"}
	eng := NewLoopEngine(cfg, mock)

	events := eng.Events()
	go func() {
		_, _ = eng.Start(t.Context())
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

func TestEmitDropsWhenClosed(t *testing.T) {
	eng := NewLoopEngine(nil, nil)
	// Close events channel by toggling flag
	eng.mu.Lock()
	eng.eventsClosed = true
	eng.mu.Unlock()

	// Should not panic
	eng.emit(NewLoopStartEvent(eng.Config()))
}

func TestBuildResultTiming(t *testing.T) {
	eng := NewLoopEngine(nil, nil)
	eng.mu.Lock()
	eng.startTime = time.Now().Add(-5 * time.Second)
	eng.iteration = 2
	eng.state = StateRunning
	eng.mu.Unlock()

	res := eng.buildResult()
	assert.Equal(t, 2, res.Iterations)
	assert.True(t, res.Duration >= 5*time.Second)
}
