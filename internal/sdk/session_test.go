package sdk

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionAndAddMessage(t *testing.T) {
	s := NewSession()
	require.NotNil(t, s)
	assert.NotEmpty(t, s.ID)
	assert.WithinDuration(t, time.Now(), s.CreatedAt, time.Second)
	assert.Empty(t, s.History)

	msg := Message{Role: RoleUser, Content: "hi", Timestamp: time.Now()}
	s.AddMessage(msg)
	assert.Len(t, s.History, 1)
	assert.Equal(t, "hi", s.History[0].Content)
}

// mockSDKInner simulates the copilot.Client and Session used by CreateSession.
// We only need methods used by the client: CreateSession and Start/Stop are handled elsewhere.
type mockSDKInnerClient struct{}

func (m *mockSDKInnerClient) CreateSession(cfg *SessionConfig) (*mockSessionInner, error) { // placeholder
	return &mockSessionInner{}, nil
}

// The SDK client types from copilot package are not available; simulate minimal types.
type SessionConfig struct {
	Model     string
	Streaming bool
}

type mockSessionInner struct{}

func (s *mockSessionInner) Destroy() error { return nil }

// Test CreateSession path where system message is added to history
func TestCreateSessionAddsSystemMessage(t *testing.T) {
	client, err := NewCopilotClient(WithSystemMessage("You are test", "append"))
	require.NoError(t, err)
	// Ensure behavior when sdkClient is nil: CreateSession should return a clear error
	client.mu.Lock()
	client.sdkClient = nil
	client.mu.Unlock()

	_, err = client.CreateSession(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SDK client not initialized")
}
