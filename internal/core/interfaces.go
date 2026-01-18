// Package core provides interfaces for dependency injection.

package core

import (
	"context"

	"github.com/JanDeDobbeleer/copilot-ralph/internal/sdk"
)

// SDKClient defines the interface for the Copilot SDK client.
// This interface abstracts the SDK implementation for testability.
type SDKClient interface {
	// Start initializes the SDK client.
	Start() error
	// Stop closes the SDK client and releases resources.
	Stop() error
	// CreateSession creates a new SDK session.
	// The implementation should initialize any SDK session resources and return an error if it fails.
	CreateSession(ctx context.Context) error
	// DestroySession destroys the current session.
	DestroySession(ctx context.Context) error
	// SendPrompt sends a prompt to the AI and returns an event stream.
	SendPrompt(ctx context.Context, prompt string) (<-chan sdk.Event, error)
	// Model returns the configured AI model name.
	Model() string
}
