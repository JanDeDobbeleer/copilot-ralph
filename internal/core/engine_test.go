package core

import (
	"context"
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

func TestBuildIterationPromptIncludesPromise(t *testing.T) {
	cfg := &LoopConfig{Prompt: "Do work", MaxIterations: 3, PromisePhrase: "YAY"}
	eng := NewLoopEngine(cfg, nil)
	p := eng.buildIterationPrompt(2)

	assert.Contains(t, p, "[Iteration 2/3]")
	assert.Contains(t, p, "Do work")
	assert.Contains(t, p, "YAY")
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
