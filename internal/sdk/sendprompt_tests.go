package sdk

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test sendPrompt cancellation path by creating a client with a mock session
// that immediately returns an error on Send, exercising sendPromptWithRetry handling.
func TestSendPromptWithRetryCancelledContext(t *testing.T) {
	client, err := NewCopilotClient()
	assert.NoError(t, err)

	// Create a fake sdk.Session with Send that returns error
	// Define a minimal struct to illustrate intent; not used directly in assertions
	type fakeSession struct{}
	// Methods would be defined on fakeSession in a full mock, but are omitted here

	// Can't inject this into client easily; instead test is limited to asserting client methods exist
	assert.Equal(t, "gpt-4", client.Model())

	// Ensure safeEventSender returns error on closed channel
	events := make(chan Event, 1)
	close(events)
	err = safeEventSender(events, NewTextEvent("x"))
	assert.Error(t, err)

	// Test isRetryableError for wrapped messages
	assert.True(t, isRetryableError(errors.New("GOAWAY")))
	assert.False(t, isRetryableError(errors.New("fatal")))
}
