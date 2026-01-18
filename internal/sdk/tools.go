// Package sdk provides tool types for Copilot SDK integration.

package sdk

// ToolCall represents a tool invocation request from the assistant.
type ToolCall struct {
	Parameters map[string]any
	ID         string
	Name       string
}
