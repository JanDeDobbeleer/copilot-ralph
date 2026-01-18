// Package core provides tests for the loop engine.

package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/JanDeDobbeleer/copilot-ralph/internal/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSDKClient is a mock implementation of SDKClient for testing.
type MockSDKClient struct {
	StopError           error
	SendPromptError     error
	DestroySessionError error
	CreateSessionError  error
	StartError          error
	ResponseText        string
	model               string
	PromisePhrase       string
	ToolCalls           []sdk.ToolCall
	mu                  sync.Mutex
	hasSession          bool
	started             bool
	SimulatePromise     bool
}

// NewMockSDKClient creates a new mock SDK client.
func NewMockSDKClient() *MockSDKClient {
	return &MockSDKClient{
		model: "mock-model",
	}
}

// Start implements SDKClient.
func (m *MockSDKClient) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.StartError != nil {
		return m.StartError
	}
	m.started = true
	return nil
}

// Stop implements SDKClient.
func (m *MockSDKClient) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.StopError != nil {
		return m.StopError
	}
	m.started = false
	return nil
}

// CreateSession implements SDKClient.
func (m *MockSDKClient) CreateSession(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.CreateSessionError != nil {
		return m.CreateSessionError
	}
	m.hasSession = true
	return nil
}

// DestroySession implements SDKClient.
func (m *MockSDKClient) DestroySession(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.DestroySessionError != nil {
		return m.DestroySessionError
	}
	m.hasSession = false
	return nil
}

// SendPrompt implements SDKClient.
func (m *MockSDKClient) SendPrompt(ctx context.Context, prompt string) (<-chan sdk.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.SendPromptError != nil {
		return nil, m.SendPromptError
	}

	events := make(chan sdk.Event, 10)

	go func() {
		defer close(events)

		// Check for cancellation
		select {
		case <-ctx.Done():
			events <- sdk.NewErrorEvent(ctx.Err())
			return
		default:
		}

		// Send text response
		responseText := m.ResponseText
		if responseText == "" {
			responseText = "Mock response"
		}

		// If simulating promise, include it in response
		if m.SimulatePromise && m.PromisePhrase != "" {
			responseText = fmt.Sprintf("%s <promise>%s</promise>", responseText, m.PromisePhrase)
		}

		events <- sdk.NewTextEvent(responseText, false)

		// Send any tool calls
		for _, tc := range m.ToolCalls {
			events <- sdk.NewToolCallEvent(tc)
			// Also send a tool result event to simulate completed tool execution
			events <- sdk.NewToolResultEvent(tc, "Mock tool result", nil)
		}
	}()

	return events, nil
}

// Model implements SDKClient.
func (m *MockSDKClient) Model() string {
	return m.model
}

// SlowMockSDKClient wraps MockSDKClient to add delays for testing cancellation.
type SlowMockSDKClient struct {
	*MockSDKClient
	delay time.Duration
}

// SendPrompt implements SDKClient with a delay.
func (m *SlowMockSDKClient) SendPrompt(ctx context.Context, prompt string) (<-chan sdk.Event, error) {
	events := make(chan sdk.Event, 10)

	go func() {
		defer close(events)

		// Add delay to allow cancellation
		select {
		case <-ctx.Done():
			events <- sdk.NewErrorEvent(ctx.Err())
			return
		case <-time.After(m.delay):
		}

		// Check for cancellation again
		select {
		case <-ctx.Done():
			events <- sdk.NewErrorEvent(ctx.Err())
			return
		default:
		}

		events <- sdk.NewTextEvent("Slow response", false)
	}()

	return events, nil
}

// TestLoopState tests the LoopState type.
func TestLoopState(t *testing.T) {
	tests := []struct {
		name       string
		state      LoopState
		wantString string
	}{
		{
			name:       "idle state",
			state:      StateIdle,
			wantString: "idle",
		},
		{
			name:       "running state",
			state:      StateRunning,
			wantString: "running",
		},
		{
			name:       "complete state",
			state:      StateComplete,
			wantString: "complete",
		},
		{
			name:       "failed state",
			state:      StateFailed,
			wantString: "failed",
		},
		{
			name:       "cancelled state",
			state:      StateCancelled,
			wantString: "cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantString, tt.state.String())
		})
	}
}

// TestNewLoopEngine tests LoopEngine creation.
func TestNewLoopEngine(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		engine := NewLoopEngine(nil, nil)

		require.NotNil(t, engine)
		assert.Equal(t, StateIdle, engine.State())
		assert.NotNil(t, engine.Config())
		assert.Equal(t, "I'm special!", engine.Config().PromisePhrase)
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &LoopConfig{
			Prompt:        "Test prompt",
			MaxIterations: 5,
			PromisePhrase: "Task complete",
		}
		mockSDK := NewMockSDKClient()
		engine := NewLoopEngine(config, mockSDK)

		require.NotNil(t, engine)
		assert.Equal(t, StateIdle, engine.State())
		assert.Equal(t, "Test prompt", engine.Config().Prompt)
		assert.Equal(t, 5, engine.Config().MaxIterations)
		assert.Equal(t, "Task complete", engine.Config().PromisePhrase)
	})

	t.Run("events channel is available", func(t *testing.T) {
		engine := NewLoopEngine(nil, nil)
		events := engine.Events()

		require.NotNil(t, events)
	})
}

// TestPromiseDetection tests the promise detection function.
func TestPromiseDetection(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		promise  string
		expected bool
	}{
		{
			name:     "exact match",
			text:     "<promise>I'm done!</promise>",
			promise:  "I'm done!",
			expected: true,
		},
		{
			name:     "case insensitive match",
			text:     "<promise>IM DONE!</promise>",
			promise:  "I'm done!",
			expected: false,
		},
		{
			name:     "embedded in text",
			text:     "The task is complete and <promise>I'm done!</promise>",
			promise:  "I'm done!",
			expected: true,
		},
		{
			name:     "not found",
			text:     "Still working on it",
			promise:  "I'm done!",
			expected: false,
		},
		{
			name:     "partial match should not match",
			text:     "I'm don",
			promise:  "I'm done!",
			expected: false,
		},
		{
			name:     "task complete phrase",
			text:     "<promise>Task complete</promise>",
			promise:  "Task complete",
			expected: true,
		},
		{
			name:     "task complete with extra text",
			text:     "All work finished. <promise>task complete</promise>",
			promise:  "task complete",
			expected: true,
		},
		{
			name:     "empty text",
			text:     "",
			promise:  "I'm done!",
			expected: false,
		},
		{
			name:     "empty promise",
			text:     "I'm done!",
			promise:  "",
			expected: false,
		},
		{
			name:     "whitespace only text",
			text:     "   ",
			promise:  "I'm done!",
			expected: false,
		},
		{
			name:     "promise with extra whitespace",
			text:     "<promise>Im   done</promise>",
			promise:  "I'm done!",
			expected: false,
		},
		{
			name:     "finished phrase",
			text:     "The task is <promise>finished</promise>.",
			promise:  "finished",
			expected: true,
		},
		{
			name:     "multiline text",
			text:     "Line 1\nLine 2\n<promise>I'm done!</promise>\nLine 4",
			promise:  "I'm done!",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectPromise(tt.text, tt.promise)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLoopEngine_StateTransitions tests state machine transitions.
func TestLoopEngine_StateTransitions(t *testing.T) {
	t.Run("initial state is idle", func(t *testing.T) {
		engine := NewLoopEngine(nil, nil)
		assert.Equal(t, StateIdle, engine.State())
	})

	t.Run("start transitions to running then complete with promise", func(t *testing.T) {
		mockSDK := NewMockSDKClient()
		mockSDK.SimulatePromise = true
		mockSDK.PromisePhrase = "I'm done!"

		config := &LoopConfig{
			Prompt:        "Test task",
			MaxIterations: 5,
			PromisePhrase: "I'm done!",
		}
		engine := NewLoopEngine(config, mockSDK)

		result, err := engine.Start(context.Background())

		require.NoError(t, err)
		assert.Equal(t, StateComplete, engine.State())
		assert.Equal(t, StateComplete, result.State)
	})

	t.Run("cancel transitions to cancelled", func(t *testing.T) {
		// Use a slow mock that blocks until cancelled
		mockSDK := &SlowMockSDKClient{
			MockSDKClient: NewMockSDKClient(),
			delay:         500 * time.Millisecond, // Slow responses
		}
		config := &LoopConfig{
			Prompt:        "Test task",
			MaxIterations: 100,
			PromisePhrase: "never found",
		}
		engine := NewLoopEngine(config, mockSDK)

		// Start in goroutine
		ctx, cancel := context.WithCancel(context.Background())
		resultCh := make(chan struct {
			result *LoopResult
			err    error
		})
		go func() {
			result, err := engine.Start(ctx)
			resultCh <- struct {
				result *LoopResult
				err    error
			}{result, err}
		}()

		// Wait a bit then cancel
		time.Sleep(100 * time.Millisecond)
		cancel()

		// Get result
		r := <-resultCh

		assert.ErrorIs(t, r.err, ErrLoopCancelled)
		assert.Equal(t, StateCancelled, engine.State())
		assert.Equal(t, StateCancelled, r.result.State)
	})
}

// TestLoopEngine_MaxIterations tests max iterations handling.
func TestLoopEngine_MaxIterations(t *testing.T) {
	mockSDK := NewMockSDKClient()
	mockSDK.ResponseText = "Working on it..."

	config := &LoopConfig{
		Prompt:        "Test task",
		MaxIterations: 3,
		PromisePhrase: "never found",
	}
	engine := NewLoopEngine(config, mockSDK)

	result, err := engine.Start(context.Background())

	// Max iterations now completes normally, not as a failure
	require.NoError(t, err)
	assert.Equal(t, StateComplete, engine.State())
	assert.Equal(t, StateComplete, result.State)
	assert.Equal(t, 3, result.Iterations)
}

// TestLoopEngine_Timeout tests timeout handling.
func TestLoopEngine_Timeout(t *testing.T) {
	// Use slow mock to ensure timeout triggers before max iterations
	mockSDK := &SlowMockSDKClient{
		MockSDKClient: NewMockSDKClient(),
		delay:         100 * time.Millisecond, // Each iteration takes 100ms
	}

	config := &LoopConfig{
		Prompt:        "Test task",
		MaxIterations: 1000,                   // Very high limit - should timeout first
		Timeout:       150 * time.Millisecond, // Timeout after ~1.5 iterations
		PromisePhrase: "never found",
	}
	engine := NewLoopEngine(config, mockSDK)

	result, err := engine.Start(context.Background())

	assert.ErrorIs(t, err, ErrLoopTimeout)
	assert.Equal(t, StateFailed, engine.State())
	assert.NotNil(t, result)
}

// TestLoopEngine_DryRun tests dry run mode.
func TestLoopEngine_DryRun(t *testing.T) {
	config := &LoopConfig{
		Prompt:        "Test task",
		MaxIterations: 2,
		PromisePhrase: "never found",
		DryRun:        true,
	}
	engine := NewLoopEngine(config, nil) // No SDK for dry run

	result, err := engine.Start(context.Background())

	// Max iterations now completes normally
	require.NoError(t, err)
	assert.Equal(t, StateComplete, result.State)
	assert.Equal(t, 2, result.Iterations)
}

// TestLoopEngine_SDKErrors tests SDK error handling.
func TestLoopEngine_SDKErrors(t *testing.T) {
	t.Run("SDK start error", func(t *testing.T) {
		mockSDK := NewMockSDKClient()
		mockSDK.StartError = errors.New("sdk start failed")

		engine := NewLoopEngine(nil, mockSDK)
		result, err := engine.Start(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start SDK")
		assert.Equal(t, StateFailed, result.State)
	})

	t.Run("SDK create session error", func(t *testing.T) {
		mockSDK := NewMockSDKClient()
		mockSDK.CreateSessionError = errors.New("session creation failed")

		engine := NewLoopEngine(nil, mockSDK)
		result, err := engine.Start(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create SDK session")
		assert.Equal(t, StateFailed, result.State)
	})

	t.Run("SDK send prompt error", func(t *testing.T) {
		mockSDK := NewMockSDKClient()
		mockSDK.SendPromptError = errors.New("prompt send failed")

		config := &LoopConfig{
			Prompt:        "Test",
			MaxIterations: 5,
			PromisePhrase: "done",
		}
		engine := NewLoopEngine(config, mockSDK)
		result, err := engine.Start(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send prompt")
		assert.Equal(t, StateFailed, result.State)
	})
}

// TestLoopResult tests LoopResult methods.
func TestLoopResult(t *testing.T) {
	// Just verify result struct can be created and used
	result := &LoopResult{
		State:      StateComplete,
		Iterations: 5,
		Duration:   30 * time.Second,
	}

	assert.Equal(t, StateComplete, result.State)
	assert.Equal(t, 5, result.Iterations)
	assert.Equal(t, 30*time.Second, result.Duration)
}

// TestDefaultLoopConfig tests default configuration values.
func TestDefaultLoopConfig(t *testing.T) {
	config := DefaultLoopConfig()

	assert.Equal(t, 10, config.MaxIterations)
	assert.Equal(t, 30*time.Minute, config.Timeout)
	assert.Equal(t, "I'm special!", config.PromisePhrase)
	assert.Equal(t, "gpt-4", config.Model)
	assert.Equal(t, ".", config.WorkingDir)
}

// TestLoopEventTypes tests event type creation.
func TestLoopEventTypes(t *testing.T) {
	t.Run("LoopStartEvent", func(t *testing.T) {
		config := &LoopConfig{Prompt: "test"}
		event := NewLoopStartEvent(config)

		assert.Equal(t, config, event.Config)
	})

	t.Run("LoopCompleteEvent", func(t *testing.T) {
		result := &LoopResult{State: StateComplete}
		event := NewLoopCompleteEvent(result)

		assert.Equal(t, result, event.Result)
	})

	t.Run("LoopFailedEvent", func(t *testing.T) {
		err := errors.New("test error")
		result := &LoopResult{State: StateFailed}
		event := NewLoopFailedEvent(err, result)

		assert.Equal(t, err, event.Error)
		assert.Equal(t, result, event.Result)
	})

	t.Run("LoopCancelledEvent", func(t *testing.T) {
		result := &LoopResult{State: StateCancelled}
		event := NewLoopCancelledEvent(result)

		assert.Equal(t, result, event.Result)
	})

	t.Run("IterationStartEvent", func(t *testing.T) {
		event := NewIterationStartEvent(3, 10)

		assert.Equal(t, 3, event.Iteration)
		assert.Equal(t, 10, event.MaxIterations)
	})

	t.Run("IterationCompleteEvent", func(t *testing.T) {
		event := NewIterationCompleteEvent(2, time.Second)

		assert.Equal(t, 2, event.Iteration)
		assert.Equal(t, time.Second, event.Duration)
	})

	t.Run("AIResponseEvent", func(t *testing.T) {
		event := NewAIResponseEvent("Hello", 1)

		assert.Equal(t, "Hello", event.Text)
		assert.Equal(t, 1, event.Iteration)
	})

	t.Run("ToolExecutionEvent", func(t *testing.T) {
		params := map[string]any{"key": "value"}
		event := NewToolExecutionEvent("read_file", params, "content", nil, time.Millisecond, 2)

		assert.Equal(t, "read_file", event.ToolName)
		assert.Equal(t, params, event.Parameters)
		assert.Equal(t, "content", event.Result)
		assert.Nil(t, event.Error)
		assert.Equal(t, time.Millisecond, event.Duration)
		assert.Equal(t, 2, event.Iteration)
	})

	t.Run("PromiseDetectedEvent", func(t *testing.T) {
		event := NewPromiseDetectedEvent("I'm done!", "ai_response", 5)

		assert.Equal(t, "I'm done!", event.Phrase)
		assert.Equal(t, "ai_response", event.Source)
		assert.Equal(t, 5, event.Iteration)
	})

	t.Run("ErrorEvent", func(t *testing.T) {
		err := errors.New("test error")
		event := NewErrorEvent(err, 2, true)

		assert.Equal(t, err, event.Error)
		assert.Equal(t, 2, event.Iteration)
		assert.True(t, event.Recoverable)
	})
}

// TestLoopEngine_Iteration tests iteration counter.
func TestLoopEngine_Iteration(t *testing.T) {
	engine := NewLoopEngine(nil, nil)

	assert.Equal(t, 0, engine.Iteration())
}
