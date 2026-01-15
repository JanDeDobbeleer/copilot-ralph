// Package sdk provides session management for Copilot SDK integration.

package sdk

import (
	"context"
	"fmt"
	"sync"
	"time"

	copilot "github.com/github/copilot-sdk/go"
	"github.com/google/uuid"
)

// MessageRole represents the role of a message sender.
type MessageRole string

const (
	// RoleUser indicates a user message.
	RoleUser MessageRole = "user"
	// RoleAssistant indicates an assistant message.
	RoleAssistant MessageRole = "assistant"
)

// Message represents a message in the conversation history.
type Message struct {
	// Role indicates who sent the message.
	Role MessageRole
	// Content contains the message text.
	Content string
	// ToolCalls contains any tool calls in this message.
	ToolCalls []ToolCall
	// Timestamp indicates when the message was created.
	Timestamp time.Time
}

// Session represents an active Copilot session.
type Session struct {
	// ID is the unique session identifier.
	ID string
	// CreatedAt indicates when the session was created.
	CreatedAt time.Time
	// History contains the conversation history.
	History []Message

	mu sync.RWMutex
}

// NewSession creates a new session with a unique ID.
func NewSession() *Session {
	return &Session{
		ID:        uuid.New().String(),
		CreatedAt: time.Now(),
		History:   make([]Message, 0),
	}
}

// AddMessage adds a message to the session history.
func (s *Session) AddMessage(msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.History = append(s.History, msg)
}

// CreateSession creates a new Copilot session.
// It initializes the session and registers it with the client.
func (c *CopilotClient) CreateSession(ctx context.Context) (*Session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		if err := c.startLocked(); err != nil {
			return nil, fmt.Errorf("failed to start client: %w", err)
		}
	}

	if c.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Build session config for the SDK
	sessionConfig := &copilot.SessionConfig{
		Model:     c.model,
		Streaming: c.streaming,
	}

	// Configure system message if provided
	if c.systemMessage != "" {
		sessionConfig.SystemMessage = &copilot.SystemMessageConfig{
			Mode:    c.systemMessageMode,
			Content: c.systemMessage,
		}
	}

	// Create SDK session
	sdkSession, err := c.sdkClient.CreateSession(sessionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDK session: %w", err)
	}

	// Create local session wrapper
	session := NewSession()

	// Add system message to local history if configured
	if c.systemMessage != "" {
		session.AddMessage(Message{
			Role:      RoleUser,
			Content:   c.systemMessage,
			Timestamp: time.Now(),
		})
	}

	c.session = session
	c.sdkSession = sdkSession
	return session, nil
}

// DestroySession destroys the current session and cleans up resources.
func (c *CopilotClient) DestroySession(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session == nil && c.sdkSession == nil {
		return nil
	}

	// Destroy SDK session if it exists
	if c.sdkSession != nil {
		_ = c.sdkSession.Destroy()
		c.sdkSession = nil
	}

	c.session = nil
	return nil
}
