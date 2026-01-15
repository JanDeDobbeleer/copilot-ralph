// Package sdk provides tool types for Copilot SDK integration.

package sdk

// ToolCall represents a tool invocation request from the assistant.
type ToolCall struct {
	// ID is the unique identifier for this tool call.
	ID string
	// Name is the name of the tool to invoke.
	Name string
	// Parameters contains the parameters for the tool.
	Parameters map[string]interface{}
}
