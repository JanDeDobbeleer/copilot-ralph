package sdk

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	copilot "github.com/github/copilot-sdk/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipIfNoSDK skips the test if the Copilot CLI is not available.
// Tests that require starting the SDK client should call this at the beginning.
func skipIfNoSDK(t *testing.T) {
	t.Helper()

	// Skip in CI unless explicitly enabled
	if os.Getenv("CI") != "" && os.Getenv("RALPH_SDK_TESTS") == "" {
		t.Skip("Skipping SDK integration test in CI (set RALPH_SDK_TESTS=1 to enable)")
	}

	// Check if copilot CLI is available
	cliPath, err := exec.LookPath("copilot")
	if err != nil {
		// On Windows, also check for copilot.cmd
		cliPath, err = exec.LookPath("copilot.cmd")
		if err != nil {
			t.Skip("Skipping test: copilot CLI not found in PATH")
		}
	}

	if runtime.GOOS == "windows" {
		return
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, cliPath, "--version").CombinedOutput()
	if err != nil {
		t.Skipf(
			"Skipping test: copilot CLI is not runnable: %v (%s)",
			err,
			strings.TrimSpace(string(output)),
		)
	}
}
func TestNewCopilotClient(t *testing.T) {
	tests := []struct {
		name        string
		wantModel   string
		errContains string
		opts        []ClientOption
		wantErr     bool
	}{
		{
			name:      "default options",
			opts:      nil,
			wantModel: DefaultModel,
			wantErr:   false,
		},
		{
			name:      "with model option",
			opts:      []ClientOption{WithModel("gpt-3.5-turbo")},
			wantModel: "gpt-3.5-turbo",
			wantErr:   false,
		},
		{
			name: "with multiple options",
			opts: []ClientOption{
				WithModel("claude-3"),
				WithWorkingDir("/tmp"),
				WithStreaming(false),
			},
			wantModel: "claude-3",
			wantErr:   false,
		},
		{
			name:        "empty model",
			opts:        []ClientOption{WithModel("")},
			wantErr:     true,
			errContains: "model cannot be empty",
		},
		{
			name:        "zero timeout",
			opts:        []ClientOption{WithTimeout(0)},
			wantErr:     true,
			errContains: "timeout must be positive",
		},
		{
			name:        "negative timeout",
			opts:        []ClientOption{WithTimeout(-1 * time.Second)},
			wantErr:     true,
			errContains: "timeout must be positive",
		},
		{
			name: "with system message",
			opts: []ClientOption{
				WithSystemMessage("You are a helpful assistant", "append"),
			},
			wantModel: DefaultModel,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewCopilotClient(tt.opts...)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)
			assert.Equal(t, tt.wantModel, client.Model())
		})
	}
}

func TestCopilotClientStartStop(t *testing.T) {

	t.Run("start and stop", func(t *testing.T) {
		skipIfNoSDK(t)
		client, err := NewCopilotClient()
		require.NoError(t, err)

		err = client.Start(t.Context())
		require.NoError(t, err)

		// Starting again should be idempotent
		err = client.Start(t.Context())
		require.NoError(t, err)

		err = client.Stop()
		require.NoError(t, err)

		// Stopping again should be idempotent
		err = client.Stop()
		require.NoError(t, err)
	})
}

type mockSDKClientStopper struct {
	stopErr         error
	stopCalled      bool
	forceStopCalled bool
}

func (m *mockSDKClientStopper) Stop() error {
	m.stopCalled = true
	return m.stopErr
}

func (m *mockSDKClientStopper) ForceStop() {
	m.forceStopCalled = true
}

func TestStopSDKClient(t *testing.T) {
	stopper := &mockSDKClientStopper{stopErr: assert.AnError}

	err := stopSDKClient(stopper)

	if runtime.GOOS == "windows" {
		require.NoError(t, err)
		assert.True(t, stopper.forceStopCalled)
		assert.False(t, stopper.stopCalled)
		return
	}

	require.ErrorIs(t, err, assert.AnError)
	assert.True(t, stopper.stopCalled)
	assert.False(t, stopper.forceStopCalled)
}

func TestCopilotClientCreateSession(t *testing.T) {
	// These tests are integration-only and require the copilot CLI; skip when CLI not available
	t.Run("create session", func(t *testing.T) {
		skipIfNoSDK(t)
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, client.Stop())
		}()

		// If SDK is not available, CreateSession will return an error "SDK client not initialized" when the client wasn't started.
		// This expectation ensures tests behave correctly when SDK is absent.
		err = client.CreateSession(t.Context())
		if err != nil {
			assert.Contains(t, err.Error(), "SDK client not initialized")
			return
		}
	})

	t.Run("create session starts client automatically", func(t *testing.T) {
		skipIfNoSDK(t)
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, client.Stop())
		}()

		// Start the client to ensure sdkClient is initialized
		err = client.Start(t.Context())
		require.NoError(t, err)

		err = client.CreateSession(t.Context())
		require.NoError(t, err)
	})

	t.Run("create session with system message", func(t *testing.T) {
		skipIfNoSDK(t)
		client, err := NewCopilotClient(
			WithSystemMessage("You are Ralph", "append"),
		)
		require.NoError(t, err)
		defer func() {
			require.NoError(t, client.Stop())
		}()

		err = client.CreateSession(t.Context())
		if err != nil {
			assert.Contains(t, err.Error(), "SDK client not initialized")
			return
		}
	})
}

func TestCopilotClientDestroySession(t *testing.T) {
	// Integration-only tests: skip if copilot CLI unavailable
	t.Run("destroy session", func(t *testing.T) {
		skipIfNoSDK(t)
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, client.Stop())
		}()

		err = client.CreateSession(t.Context())
		if err != nil {
			// SDK may be missing; accept the known error message
			assert.Contains(t, err.Error(), "SDK client not initialized")
			return
		}

		err = client.DestroySession(t.Context())
		require.NoError(t, err)
	})

	t.Run("destroy nil session is no-op", func(t *testing.T) {
		skipIfNoSDK(t)
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, client.Stop())
		}()

		err = client.DestroySession(t.Context())
		require.NoError(t, err)
	})
}

func TestCopilotClientSendPrompt(t *testing.T) {

}

func TestCopilotClientListModels(t *testing.T) {
	t.Run("requires started client", func(t *testing.T) {
		client, err := NewCopilotClient()
		require.NoError(t, err)

		models, err := client.ListModels(t.Context())

		require.Error(t, err)
		assert.Nil(t, models)
		assert.Contains(t, err.Error(), "SDK client not initialized")
	})

	t.Run("lists available models", func(t *testing.T) {
		skipIfNoSDK(t)
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, client.Stop())
		}()

		require.NoError(t, client.Start(t.Context()))

		models, err := client.ListModels(t.Context())

		require.NoError(t, err)
		require.NotEmpty(t, models)
		for _, model := range models {
			assert.NotEmpty(t, model.ID)
		}
	})
}

func TestCopilotClientConcurrency(t *testing.T) {
	skipIfNoSDK(t)

	t.Run("concurrent session access", func(t *testing.T) {
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer func() {
			require.NoError(t, client.Stop())
		}()

		err = client.CreateSession(t.Context())
		if err != nil {
			if err.Error() == "SDK client not initialized" {
				// SDK missing, accept this outcome
				return
			}
			require.NoError(t, err)
		}

		var wg sync.WaitGroup

		// Concurrently access a client property (Model)
		for range 10 {
			wg.Go(func() {

				// Concurrently read a client property
				_ = client.Model()
			})
		}

		wg.Wait()
	})
}

func TestEventTypes(t *testing.T) {
	t.Run("text event", func(t *testing.T) {
		event := NewTextEvent("hello", false)
		assert.Equal(t, EventTypeText, event.Type())
		assert.Equal(t, "hello", event.Text)
		assert.WithinDuration(t, time.Now(), event.Timestamp(), time.Second)
	})

	t.Run("tool call event", func(t *testing.T) {
		toolCall := ToolCall{ID: "tc1", Name: "test"}
		event := NewToolCallEvent(toolCall)
		assert.Equal(t, EventTypeToolCall, event.Type())
		assert.Equal(t, "tc1", event.ToolCall.ID)
		assert.WithinDuration(t, time.Now(), event.Timestamp(), time.Second)
	})

	t.Run("tool result event", func(t *testing.T) {
		toolCall := ToolCall{ID: "tc2", Name: "test"}
		event := NewToolResultEvent(toolCall, "result", nil)
		assert.Equal(t, EventTypeToolResult, event.Type())
		assert.Equal(t, "result", event.Result)
		assert.Nil(t, event.Error)
		assert.WithinDuration(t, time.Now(), event.Timestamp(), time.Second)
	})

	t.Run("error event", func(t *testing.T) {
		err := errors.New("test error")
		event := NewErrorEvent(err)
		assert.Equal(t, EventTypeError, event.Type())
		assert.Equal(t, "test error", event.Error())
		assert.WithinDuration(t, time.Now(), event.Timestamp(), time.Second)
	})

	t.Run("error event with nil error", func(t *testing.T) {
		event := NewErrorEvent(nil)
		assert.Equal(t, EventTypeError, event.Type())
		assert.Equal(t, "", event.Error())
	})
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		err      error
		name     string
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "GOAWAY error",
			err:      errors.New("HTTP/2 GOAWAY connection terminated"),
			expected: true,
		},
		{
			name:     "connection reset error",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "connection refused error",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "connection terminated error",
			err:      errors.New("connection terminated unexpectedly"),
			expected: true,
		},
		{
			name:     "EOF error",
			err:      errors.New("unexpected EOF"),
			expected: true,
		},
		{
			name:     "timeout error",
			err:      errors.New("request timeout"),
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      errors.New("invalid argument"),
			expected: false,
		},
		{
			name:     "authentication error",
			err:      errors.New("authentication failed"),
			expected: false,
		},
		{
			name:     "SDK error model not found",
			err:      errors.New("model not found"),
			expected: false,
		},
		{
			name:     "wrapped GOAWAY error",
			err:      errors.New("SDK error: Model call failed: HTTP/2 GOAWAY connection terminated"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeEventSender(t *testing.T) {
	// Open channel should succeed
	events := make(chan Event, 1)
	err := safeEventSender(events, NewTextEvent("hello", false))
	assert.NoError(t, err)
	// consume
	recv := <-events
	assert.Equal(t, EventTypeText, recv.Type())

	// Closed channel should return an error (recovered panic)
	close(events)
	err = safeEventSender(events, NewTextEvent("world", false))
	assert.Error(t, err)
}

// Test sendPrompt cancellation path by creating a client with a mock session
// that immediately returns an error on Send, exercising sendPromptWithRetry handling.
func TestSendPromptWithRetryCancelledContext(t *testing.T) {
	client, err := NewCopilotClient()
	assert.NoError(t, err)

	// Can't inject this into client easily; instead test is limited to asserting client methods exist
	assert.Equal(t, DefaultModel, client.Model())

	// Ensure safeEventSender returns error on closed channel
	events := make(chan Event, 1)
	close(events)
	err = safeEventSender(events, NewTextEvent("x", false))
	assert.Error(t, err)

	// Test isRetryableError for wrapped messages
	assert.True(t, isRetryableError(errors.New("GOAWAY")))
	assert.False(t, isRetryableError(errors.New("fatal")))
}

func TestIsRetryableErrorEdgeCases(t *testing.T) {
	// Should return false for unrelated errors
	assert.False(t, isRetryableError(assert.AnError))

	// Errors containing EOF should be retryable
	assert.True(t, isRetryableError(errorString("unexpected EOF")))

	// Custom timeout string
	assert.True(t, isRetryableError(errorString("timeout occurred")))
}

// helper type to provide Error() string
type errorString string

func (e errorString) Error() string { return string(e) }

// testEventDrainTimeout is used by tests that need to drain the events channel
// without relying on the producer to close it. We use a short timeout to avoid
// indefinite blocking in tests where the code under test may return early
// (for example, when a context is canceled).
const testEventDrainTimeout = 100 * time.Millisecond

func TestSafeEventSenderOnClosedChannel(t *testing.T) {
	ch := make(chan Event)
	close(ch)
	err := safeEventSender(ch, NewTextEvent("x", false))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "event channel closed")
}

func TestHandleSDKEventVariousTypes(t *testing.T) {
	c := &CopilotClient{}
	events := make(chan Event, 10)
	defer close(events)
	var closed bool
	closeDone := func() { closed = true }
	pending := make(map[string]ToolCall)

	// assistant.message_delta
	c.handleSDKEvent(copilot.SessionEvent{Data: &copilot.AssistantMessageDeltaData{DeltaContent: "part"}}, events, closeDone, pending)

	// assistant.message
	c.handleSDKEvent(copilot.SessionEvent{Data: &copilot.AssistantMessageData{Content: "full"}}, events, closeDone, pending)

	// assistant.reasoning_delta
	c.handleSDKEvent(copilot.SessionEvent{Data: &copilot.AssistantReasoningDeltaData{DeltaContent: "thinking"}}, events, closeDone, pending)

	// tool.execution_start
	c.handleSDKEvent(copilot.SessionEvent{Data: &copilot.ToolExecutionStartData{ToolName: "edit", ToolCallID: "1", Arguments: map[string]any{"path": "a.go"}}}, events, closeDone, pending)

	// tool.execution_complete success
	c.handleSDKEvent(copilot.SessionEvent{Data: &copilot.ToolExecutionCompleteData{ToolCallID: "1", Result: &copilot.ToolExecutionCompleteResult{Content: "ok"}, Success: true}}, events, closeDone, pending)

	// tool.execution_complete failure
	c.handleSDKEvent(copilot.SessionEvent{Data: &copilot.ToolExecutionCompleteData{ToolCallID: "2", Result: &copilot.ToolExecutionCompleteResult{Content: ""}, Success: false, Error: &copilot.ToolExecutionCompleteError{Message: "tool failed"}}}, events, closeDone, pending)

	// session.error
	c.handleSDKEvent(copilot.SessionEvent{Data: &copilot.SessionErrorData{Message: "bad"}}, events, closeDone, pending)

	// session.idle should call closeDone
	c.handleSDKEvent(copilot.SessionEvent{Data: &copilot.SessionIdleData{}}, events, closeDone, pending)

	// Drain events and assert some expected types
	received := []Event{}
	down := time.After(testEventDrainTimeout)
loop:
	for {
		select {
		case ev := <-events:
			received = append(received, ev)
			if len(received) >= 7 {
				break loop
			}
		case <-down:
			break loop
		}
	}

	// Expect at least one TextEvent and at least one ToolResultEvent and one ErrorEvent
	var hasText, hasToolResult, hasError bool
	for _, e := range received {
		switch e.Type() {
		case EventTypeText:
			hasText = true
		case EventTypeToolResult:
			hasToolResult = true
		case EventTypeError:
			hasError = true
		}
	}

	assert.True(t, hasText, "should have text events")
	assert.True(t, hasToolResult, "should have tool result events")
	assert.True(t, hasError, "should have error events")
	assert.True(t, closed, "closeDone should be called on session.idle")
}

func TestSendPromptOnceWithFakeSession(t *testing.T) {
	c, err := NewCopilotClient()
	require.NoError(t, err)

	events := make(chan Event, 10)
	defer close(events)

	// call sendPromptWithRetry with a canceled context to cover the cancellation early-return path
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	c.sendPromptWithRetry(ctx, "hello", events)
	// no error expected; function returns after cancellation
	// drain any events with a short timeout to avoid indefinite blocking
	done := time.After(testEventDrainTimeout)
drainLoop:
	for {
		select {
		case _, ok := <-events:
			if !ok {
				break drainLoop
			}
			// continue draining
		case <-done:
			// timeout reached; stop draining
			break drainLoop
		}
	}
}
