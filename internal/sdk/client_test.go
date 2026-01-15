package sdk

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

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
	_, err := exec.LookPath("copilot")
	if err != nil {
		// On Windows, also check for copilot.cmd
		_, err = exec.LookPath("copilot.cmd")
		if err != nil {
			t.Skip("Skipping test: copilot CLI not found in PATH")
		}
	}
}
func TestNewCopilotClient(t *testing.T) {
	tests := []struct {
		name        string
		opts        []ClientOption
		wantModel   string
		wantErr     bool
		errContains string
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
	skipIfNoSDK(t)

	t.Run("start and stop", func(t *testing.T) {
		client, err := NewCopilotClient()
		require.NoError(t, err)

		err = client.Start()
		require.NoError(t, err)

		// Starting again should be idempotent
		err = client.Start()
		require.NoError(t, err)

		err = client.Stop()
		require.NoError(t, err)

		// Stopping again should be idempotent
		err = client.Stop()
		require.NoError(t, err)
	})
}

func TestCopilotClientCreateSession(t *testing.T) {
	skipIfNoSDK(t)

	t.Run("create session", func(t *testing.T) {
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer client.Stop()

		session, err := client.CreateSession(context.Background())
		require.NoError(t, err)
		require.NotNil(t, session)

		assert.NotEmpty(t, session.ID)
		assert.WithinDuration(t, time.Now(), session.CreatedAt, time.Second)
		assert.Empty(t, session.History)
	})

	t.Run("create session starts client automatically", func(t *testing.T) {
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer client.Stop()

		_, err = client.CreateSession(context.Background())
		require.NoError(t, err)
	})

	t.Run("create session with system message", func(t *testing.T) {
		client, err := NewCopilotClient(
			WithSystemMessage("You are Ralph", "append"),
		)
		require.NoError(t, err)
		defer client.Stop()

		session, err := client.CreateSession(context.Background())
		require.NoError(t, err)

		// System message is added to history as user message
		require.Len(t, session.History, 1)
		assert.Equal(t, RoleUser, session.History[0].Role)
		assert.Equal(t, "You are Ralph", session.History[0].Content)
	})
}

func TestCopilotClientDestroySession(t *testing.T) {
	skipIfNoSDK(t)

	t.Run("destroy session", func(t *testing.T) {
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer client.Stop()

		_, err = client.CreateSession(context.Background())
		require.NoError(t, err)

		err = client.DestroySession(context.Background())
		require.NoError(t, err)
	})

	t.Run("destroy nil session is no-op", func(t *testing.T) {
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer client.Stop()

		err = client.DestroySession(context.Background())
		require.NoError(t, err)
	})
}

func TestCopilotClientSendPrompt(t *testing.T) {
	skipIfNoSDK(t)

	t.Run("send prompt without session", func(t *testing.T) {
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer client.Stop()

		_, err = client.SendPrompt(context.Background(), "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no active session")
	})

	t.Run("send prompt with session", func(t *testing.T) {
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer client.Stop()

		_, err = client.CreateSession(context.Background())
		require.NoError(t, err)

		events, err := client.SendPrompt(context.Background(), "Hello, world!")
		require.NoError(t, err)
		require.NotNil(t, events)

		// Collect all events
		var receivedEvents []Event
		for event := range events {
			receivedEvents = append(receivedEvents, event)
		}

		// Should have at least text and response complete events
		require.GreaterOrEqual(t, len(receivedEvents), 2)

		// Check for text event
		hasText := false
		hasComplete := false
		for _, e := range receivedEvents {
			if e.Type() == EventTypeText {
				hasText = true
			}
			if e.Type() == EventTypeResponseComplete {
				hasComplete = true
			}
		}
		assert.True(t, hasText, "should have text event")
		assert.True(t, hasComplete, "should have response complete event")
	})

	t.Run("send prompt records in history", func(t *testing.T) {
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer client.Stop()

		session, err := client.CreateSession(context.Background())
		require.NoError(t, err)

		events, err := client.SendPrompt(context.Background(), "Test prompt")
		require.NoError(t, err)

		// Drain events
		for range events {
		}

		// Check that messages were added to history
		require.GreaterOrEqual(t, len(session.History), 2)

		// First should be user message
		assert.Equal(t, RoleUser, session.History[0].Role)
		assert.Equal(t, "Test prompt", session.History[0].Content)

		// Second should be assistant message
		assert.Equal(t, RoleAssistant, session.History[1].Role)
	})

	t.Run("send prompt with cancelled context", func(t *testing.T) {
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer client.Stop()

		_, err = client.CreateSession(context.Background())
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		events, err := client.SendPrompt(ctx, "Test")
		require.NoError(t, err)

		// When context is cancelled before processing, the channel should close
		// without sending an error event (caller already knows context is cancelled)
		var receivedEvents []Event
		for event := range events {
			receivedEvents = append(receivedEvents, event)
		}

		// Should have no events or minimal events (channel closes quickly)
		// This is the expected behavior - cancelled context means clean exit
		// The caller can check ctx.Err() to know it was cancelled
		assert.True(t, len(receivedEvents) == 0 || func() bool {
			// If there are events, they should only be error events for cancelled context
			for _, e := range receivedEvents {
				if errEv, ok := e.(*ErrorEvent); ok {
					if errors.Is(errEv.Err, context.Canceled) {
						return true
					}
				}
			}
			return len(receivedEvents) == 0
		}())
	})
}

func TestCopilotClientConcurrency(t *testing.T) {
	skipIfNoSDK(t)

	t.Run("concurrent session access", func(t *testing.T) {
		client, err := NewCopilotClient()
		require.NoError(t, err)
		defer client.Stop()

		session, err := client.CreateSession(context.Background())
		require.NoError(t, err)

		var wg sync.WaitGroup
		errChan := make(chan error, 10)

		// Multiple goroutines accessing session
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				if session == nil {
					errChan <- errors.New("session is nil")
					return
				}

				// Just access session concurrently
				_ = session.ID
			}()
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			t.Errorf("Concurrent access error: %v", err)
		}
	})
}

func TestSession(t *testing.T) {
	t.Run("new session", func(t *testing.T) {
		session := NewSession()
		require.NotNil(t, session)

		assert.NotEmpty(t, session.ID)
		assert.WithinDuration(t, time.Now(), session.CreatedAt, time.Second)
		assert.Empty(t, session.History)
	})

	t.Run("add messages", func(t *testing.T) {
		session := NewSession()

		msg := Message{
			Role:      RoleUser,
			Content:   "Hello",
			Timestamp: time.Now(),
		}

		session.AddMessage(msg)

		require.Len(t, session.History, 1)
		assert.Equal(t, RoleUser, session.History[0].Role)
		assert.Equal(t, "Hello", session.History[0].Content)
	})
}

func TestEventTypes(t *testing.T) {
	t.Run("text event", func(t *testing.T) {
		event := NewTextEvent("hello")
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

	t.Run("response complete event", func(t *testing.T) {
		msg := Message{Role: RoleAssistant, Content: "done"}
		event := NewResponseCompleteEvent(msg)
		assert.Equal(t, EventTypeResponseComplete, event.Type())
		assert.Equal(t, RoleAssistant, event.Message.Role)
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
		name     string
		err      error
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
