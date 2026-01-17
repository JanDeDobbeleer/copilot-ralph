// Package sdk provides event types for Copilot SDK communication.

package sdk

import (
	"time"
)

// EventType represents the type of event from the Copilot SDK.
type EventType string

const (
	// EventTypeText indicates a text/streaming content event.
	EventTypeText EventType = "text"
	// EventTypeToolCall indicates a tool invocation request event.
	EventTypeToolCall EventType = "tool_call"
	// EventTypeToolResult indicates a tool execution result event.
	EventTypeToolResult EventType = "tool_result"
	// EventTypeResponseComplete indicates the response is complete.
	EventTypeResponseComplete EventType = "response_complete"
	// EventTypeError indicates an error occurred.
	EventTypeError EventType = "error"
)

// Event represents an event from the Copilot SDK.
// All event types implement this interface.
type Event interface {
	// Type returns the event type.
	Type() EventType
	// Timestamp returns when the event occurred.
	Timestamp() time.Time
}

// TextEvent represents a text/streaming content event.
type TextEvent struct {
	// Text contains the text content from the assistant.
	Text      string
	timestamp time.Time
}

// Type returns EventTypeText.
func (e *TextEvent) Type() EventType {
	return EventTypeText
}

// Timestamp returns when the event occurred.
func (e *TextEvent) Timestamp() time.Time {
	return e.timestamp
}

// NewTextEvent creates a new TextEvent with the given text.
func NewTextEvent(text string) *TextEvent {
	return &TextEvent{
		Text:      text,
		timestamp: time.Now(),
	}
}

// ToolCallEvent represents a tool invocation request from the assistant.
type ToolCallEvent struct {
	// ToolCall contains the tool call details.
	ToolCall  ToolCall
	timestamp time.Time
}

// Type returns EventTypeToolCall.
func (e *ToolCallEvent) Type() EventType {
	return EventTypeToolCall
}

// Timestamp returns when the event occurred.
func (e *ToolCallEvent) Timestamp() time.Time {
	return e.timestamp
}

// NewToolCallEvent creates a new ToolCallEvent with the given tool call.
func NewToolCallEvent(toolCall ToolCall) *ToolCallEvent {
	return &ToolCallEvent{
		ToolCall:  toolCall,
		timestamp: time.Now(),
	}
}

// ToolResultEvent represents the result of a tool execution.
type ToolResultEvent struct {
	// ToolCall contains the original tool call.
	ToolCall ToolCall
	// Result contains the tool execution result.
	Result string
	// Error contains any error that occurred during execution.
	Error     error
	timestamp time.Time
}

// Type returns EventTypeToolResult.
func (e *ToolResultEvent) Type() EventType {
	return EventTypeToolResult
}

// Timestamp returns when the event occurred.
func (e *ToolResultEvent) Timestamp() time.Time {
	return e.timestamp
}

// NewToolResultEvent creates a new ToolResultEvent with the given result.
func NewToolResultEvent(toolCall ToolCall, result string, err error) *ToolResultEvent {
	return &ToolResultEvent{
		ToolCall:  toolCall,
		Result:    result,
		Error:     err,
		timestamp: time.Now(),
	}
}

// ResponseCompleteEvent indicates that the assistant's response is complete.
type ResponseCompleteEvent struct {
	// Message contains the complete message if available.
	Message   Message
	timestamp time.Time
}

// Type returns EventTypeResponseComplete.
func (e *ResponseCompleteEvent) Type() EventType {
	return EventTypeResponseComplete
}

// Timestamp returns when the event occurred.
func (e *ResponseCompleteEvent) Timestamp() time.Time {
	return e.timestamp
}

// NewResponseCompleteEvent creates a new ResponseCompleteEvent.
func NewResponseCompleteEvent(msg Message) *ResponseCompleteEvent {
	return &ResponseCompleteEvent{
		Message:   msg,
		timestamp: time.Now(),
	}
}

// ErrorEvent represents an error that occurred during processing.
type ErrorEvent struct {
	// Err contains the error that occurred.
	Err       error
	timestamp time.Time
}

// Type returns EventTypeError.
func (e *ErrorEvent) Type() EventType {
	return EventTypeError
}

// Timestamp returns when the event occurred.
func (e *ErrorEvent) Timestamp() time.Time {
	return e.timestamp
}

// Error returns the error message.
func (e *ErrorEvent) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

// NewErrorEvent creates a new ErrorEvent with the given error.
func NewErrorEvent(err error) *ErrorEvent {
	return &ErrorEvent{
		Err:       err,
		timestamp: time.Now(),
	}
}
