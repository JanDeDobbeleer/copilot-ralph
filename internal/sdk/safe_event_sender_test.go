package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeEventSender(t *testing.T) {
	// Open channel should succeed
	events := make(chan Event, 1)
	err := safeEventSender(events, NewTextEvent("hello"))
	assert.NoError(t, err)
	// consume
	recv := <-events
	assert.Equal(t, EventTypeText, recv.Type())

	// Closed channel should return an error (recovered panic)
	close(events)
	err = safeEventSender(events, NewTextEvent("world"))
	assert.Error(t, err)
}
