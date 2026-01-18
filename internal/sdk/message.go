// Package sdk

package sdk

import "time"

// Message represents a message that can accompany response-complete events.
// It is intentionally minimal and no longer tracked in a local session object.

type Message struct {
	Timestamp time.Time
	Content   string
	ToolCalls []ToolCall
}
