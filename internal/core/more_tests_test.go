package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
