// Package core implements the Ralph loop engine and business logic.
//
// This package contains the core loop execution engine that orchestrates
// iterative AI development loops. It manages state transitions, promise
// detection, and event emission.
//
// The LoopEngine follows a state machine pattern with the following states:
//   - StateIdle: Initial state, ready to start
//   - StateRunning: Loop is executing iterations
//   - StateComplete: Successfully completed (promise detected)
//   - StateFailed: Failed due to error, timeout, or max iterations
//   - StateCancelled: Cancelled by user
//
// See specs/loop-engine.md for detailed specification.
package core

import (
	"context"
	"strings"
	"sync"
	"time"
)

// LoopState represents the current state of the loop.
type LoopState string

const (
	// StateIdle indicates the loop is ready to start.
	StateIdle LoopState = "idle"
	// StateRunning indicates the loop is executing iterations.
	StateRunning LoopState = "running"
	// StateComplete indicates the loop completed successfully.
	StateComplete LoopState = "complete"
	// StateFailed indicates the loop failed.
	StateFailed LoopState = "failed"
	// StateCancelled indicates the loop was cancelled.
	StateCancelled LoopState = "cancelled"
)

// String returns the string representation of the state.
func (s LoopState) String() string {
	return string(s)
}

// LoopConfig contains configuration for loop execution.
type LoopConfig struct {
	Prompt        string
	PromisePhrase string
	Model         string
	WorkingDir    string
	MaxIterations int
	Timeout       time.Duration
	DryRun        bool
}

// DefaultLoopConfig returns a LoopConfig with default values.
func DefaultLoopConfig() *LoopConfig {
	return &LoopConfig{
		MaxIterations: 10,
		Timeout:       30 * time.Minute,
		PromisePhrase: "I'm special!",
		Model:         "gpt-4",
		WorkingDir:    ".",
	}
}

// LoopEngine manages the execution of AI development loops.
// It coordinates with the Copilot SDK, detects completion promises,
// and handles state transitions.
type LoopEngine struct {
	startTime    time.Time
	sdk          SDKClient
	ctx          context.Context
	config       *LoopConfig
	events       chan any
	cancel       context.CancelFunc
	state        LoopState
	iteration    int
	mu           sync.RWMutex
	eventsClosed bool
}

// eventChannelBufferSize is the buffer size for the events channel.
const eventChannelBufferSize = 100

// NewLoopEngine creates a new loop engine with the given configuration.
// If sdk is nil, the engine will run in dry-run mode.
func NewLoopEngine(config *LoopConfig, sdk SDKClient) *LoopEngine {
	if config == nil {
		config = DefaultLoopConfig()
	}

	return &LoopEngine{
		config: config,
		sdk:    sdk,
		state:  StateIdle,
		events: make(chan any, eventChannelBufferSize),
	}
}

// State returns the current loop state.
func (e *LoopEngine) State() LoopState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

// BuildSystemPrompt constructs the system prompt from the embedded template.
// It replaces {{.Task}} with the actual user task and {{.Promise}} with the completion phrase.
func BuildSystemPrompt(promisePhrase string) string {
	return strings.ReplaceAll(systemPromptTemplate, "{{.Promise}}", promisePhrase)
}

// Iteration returns the current iteration number (1-based).
// Returns 0 if the loop hasn't started yet.
func (e *LoopEngine) Iteration() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.iteration
}

// Config returns the loop configuration.
func (e *LoopEngine) Config() *LoopConfig {
	return e.config
}

// Events returns a read-only channel for receiving loop events.
// Subscribers should read from this channel to receive updates.
func (e *LoopEngine) Events() <-chan any {
	return e.events
}

// LoopResult contains the outcome of loop execution.
type LoopResult struct {
	Error      error
	State      LoopState
	Iterations int
	Duration   time.Duration
}
