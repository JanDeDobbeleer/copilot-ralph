// Package sdk

package sdk

import "time"

// MessageRole and Message remain as minimal types for events and responses.
// The historical Session wrapper (with stored history) was removed.

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

// Message represents a message that can accompany response-complete events.
// It is intentionally minimal and no longer tracked in a local session object.

type Message struct {
	Timestamp time.Time
	Role      MessageRole
	Content   string
	ToolCalls []ToolCall
}
