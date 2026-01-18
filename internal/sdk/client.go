// Package sdk provides a wrapper around the GitHub Copilot SDK.
//
// This package abstracts Copilot SDK integration, providing session management,
// event handling, custom tool registration, and error handling. It provides
// a simplified interface for Ralph's needs while handling the complexity of
// the underlying SDK.
//
// See specs/sdk-integration.md for detailed specification.
package sdk

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	copilot "github.com/github/copilot-sdk/go"
)

// Default configuration values.
const (
	DefaultModel     = "gpt-4"
	DefaultLogLevel  = "info"
	DefaultTimeout   = 60 * time.Second
	DefaultStreaming = true
)

// Retry backoff durations for transient errors.
var retryBackoffs = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	5 * time.Second,
}

// isRetryableError determines if an error is transient and can be retried.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// HTTP/2 connection errors are transient
	if strings.Contains(errStr, "GOAWAY") {
		return true
	}
	if strings.Contains(errStr, "connection reset") {
		return true
	}
	if strings.Contains(errStr, "connection refused") {
		return true
	}
	if strings.Contains(errStr, "connection terminated") {
		return true
	}
	if strings.Contains(errStr, "EOF") {
		return true
	}
	if strings.Contains(errStr, "timeout") {
		return true
	}
	return false
}

// CopilotClient wraps the GitHub Copilot SDK.
// It provides session management, event handling, and tool registration.
type CopilotClient struct {
	sdkClient         *copilot.Client
	sdkSession        *copilot.Session
	model             string
	logLevel          string
	workingDir        string
	systemMessageMode string
	systemMessage     string
	timeout           time.Duration
	mu                sync.RWMutex
	streaming         bool
	started           bool
}

// clientConfig holds configuration options for the client.
type clientConfig struct {
	model             string
	logLevel          string
	workingDir        string
	systemMessageMode string
	systemMessage     string
	timeout           time.Duration
	streaming         bool
}

// ClientOption configures the CopilotClient.
type ClientOption func(*clientConfig)

// WithModel sets the AI model to use.
func WithModel(model string) ClientOption {
	return func(c *clientConfig) {
		c.model = model
	}
}

// WithLogLevel sets the logging level (debug, info, warn, error).
func WithLogLevel(level string) ClientOption {
	return func(c *clientConfig) {
		c.logLevel = level
	}
}

// WithWorkingDir sets the working directory for file operations.
func WithWorkingDir(dir string) ClientOption {
	return func(c *clientConfig) {
		c.workingDir = dir
	}
}

// WithStreaming enables or disables streaming responses.
func WithStreaming(streaming bool) ClientOption {
	return func(c *clientConfig) {
		c.streaming = streaming
	}
}

// WithSystemMessage sets the system message for the session.
// Mode can be "append" or "replace".
func WithSystemMessage(message, mode string) ClientOption {
	return func(c *clientConfig) {
		c.systemMessage = message
		c.systemMessageMode = mode
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.timeout = timeout
	}
}

// NewCopilotClient creates a new Copilot SDK client with the given options.
// It returns an error if the configuration is invalid.
func NewCopilotClient(opts ...ClientOption) (*CopilotClient, error) {
	// Default configuration
	config := &clientConfig{
		model:             DefaultModel,
		logLevel:          DefaultLogLevel,
		workingDir:        ".",
		streaming:         DefaultStreaming,
		systemMessageMode: "append",
		timeout:           DefaultTimeout,
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Validate configuration
	if config.model == "" {
		return nil, fmt.Errorf("model cannot be empty")
	}

	if config.timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive")
	}

	return &CopilotClient{
		model:             config.model,
		logLevel:          config.logLevel,
		workingDir:        config.workingDir,
		streaming:         config.streaming,
		systemMessageMode: config.systemMessageMode,
		systemMessage:     config.systemMessage,
		timeout:           config.timeout,
		started:           false,
	}, nil
}

// Start starts the client and prepares it for session creation.
// This must be called before creating sessions.
func (c *CopilotClient) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.startLocked()
}

// startLocked starts the client (must be called with lock held).
func (c *CopilotClient) startLocked() error {
	if c.started {
		return nil
	}

	// Initialize the SDK client with options
	c.sdkClient = copilot.NewClient(&copilot.ClientOptions{
		LogLevel: c.logLevel,
		Cwd:      c.workingDir,
	})

	// Start the SDK client
	if err := c.sdkClient.Start(); err != nil {
		return fmt.Errorf("failed to start SDK client: %w", err)
	}

	c.started = true
	return nil
}

// Stop stops the client and releases resources.
func (c *CopilotClient) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}

	// Destroy any active SDK session
	if c.sdkSession != nil {
		_ = c.sdkSession.Destroy()
		c.sdkSession = nil
	}

	// Stop the SDK client
	if c.sdkClient != nil {
		_ = c.sdkClient.Stop()
		c.sdkClient = nil
	}

	c.started = false
	return nil
}

// CreateSession creates a new Copilot session.
// It initializes the SDK session resources and registers them with the client.
func (c *CopilotClient) CreateSession(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sdkClient == nil {
		return fmt.Errorf("SDK client not initialized")
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
		return fmt.Errorf("failed to create SDK session: %w", err)
	}

	// Store SDK session reference; we no longer maintain a local Session wrapper
	c.sdkSession = sdkSession
	return nil
}

// DestroySession destroys the current session and cleans up resources.
func (c *CopilotClient) DestroySession(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sdkSession == nil {
		return nil
	}

	_ = c.sdkSession.Destroy()
	c.sdkSession = nil
	return nil
}

// Model returns the configured model name.
func (c *CopilotClient) Model() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.model
}

// SendPrompt sends a prompt to the Copilot SDK and returns an event stream.
// The returned channel will be closed when the response is complete.
// An error is returned if there is no active session.
// This method includes automatic retry logic for transient errors.
func (c *CopilotClient) SendPrompt(ctx context.Context, prompt string) (<-chan Event, error) {
	c.mu.Lock()
	if c.sdkSession == nil {
		c.mu.Unlock()
		return nil, fmt.Errorf("no active session")
	}

	sdkSession := c.sdkSession
	c.mu.Unlock()

	// Create event channel with buffer
	events := make(chan Event, 100)

	// Process prompt asynchronously with retry logic
	go func() {
		defer close(events)
		c.sendPromptWithRetry(ctx, sdkSession, prompt, events)
	}()

	return events, nil
}

// safeEventSender safely sends an event to a channel, recovering from panics if the channel is closed.
// Returns an error if the send failed (e.g., channel closed).
func safeEventSender(events chan<- Event, event Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// Channel was closed, ignore the panic
			err = fmt.Errorf("event channel closed")
		}
	}()

	events <- event
	return nil
}

// sendPromptWithRetry sends the prompt with automatic retry for transient errors.
func (c *CopilotClient) sendPromptWithRetry(ctx context.Context, sdkSession *copilot.Session, prompt string, events chan<- Event) {
	var lastErr error

	for attempt := 0; attempt <= len(retryBackoffs); attempt++ {
		// Check for context cancellation before each attempt
		select {
		case <-ctx.Done():
			// Don't send error event here - just return so the channel closes
			// The caller will detect cancellation via ctx.Done()
			return
		default:
		}

		// If this is a retry, wait before trying again
		if attempt > 0 {
			backoff := retryBackoffs[attempt-1]
			select {
			case <-ctx.Done():
				_ = safeEventSender(events, NewErrorEvent(ctx.Err()))
				return
			case <-time.After(backoff):
			}
		}

		// Attempt to send the prompt
		err := c.sendPromptOnce(ctx, sdkSession, prompt, events)
		if err == nil {
			// Success
			return
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			_ = safeEventSender(events, NewErrorEvent(err))
			return
		}

		// Error is retryable, will retry on next iteration
	}

	// All retries exhausted
	_ = safeEventSender(events, NewErrorEvent(fmt.Errorf("max retries exceeded: %w", lastErr)))
}

// sendPromptOnce sends the prompt once without retrying.
func (c *CopilotClient) sendPromptOnce(ctx context.Context, sdkSession *copilot.Session, prompt string, events chan<- Event) error {
	// Set up done channel to wait for session.idle
	done := make(chan struct{})
	doneOnce := &sync.Once{}
	closeDone := func() {
		doneOnce.Do(func() {
			close(done)
		})
	}
	var responseContent string
	var sessionErr error
	pendingToolCalls := make(map[string]ToolCall)

	// Subscribe to SDK session events
	unsubscribe := sdkSession.On(func(event copilot.SessionEvent) {
		// Check if context is cancelled before processing events
		select {
		case <-ctx.Done():
			// Context cancelled, close done channel to unblock and stop processing
			closeDone()
			return
		default:
		}

		if event.Type == "session.error" && event.Data.Message != nil {
			sessionErr = fmt.Errorf("SDK error: %s", *event.Data.Message)
		}
		c.handleSDKEvent(event, events, &responseContent, closeDone, pendingToolCalls)
	})
	defer unsubscribe()

	// Send the message
	_, err := sdkSession.Send(copilot.MessageOptions{
		Prompt: prompt,
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Wait for session to become idle or context cancellation
	select {
	case <-ctx.Done():
		// Abort the session and close done to unblock any waiting
		go func() {
			_ = sdkSession.Abort()
		}()
		closeDone()
		return ctx.Err()
	case <-done:
		// Response complete - check for session error
		if sessionErr != nil {
			return sessionErr
		}
	}

	// Signal completion
	_ = safeEventSender(events, NewResponseCompleteEvent(Message{
		Role:      RoleAssistant,
		Content:   responseContent,
		Timestamp: time.Now(),
	}))

	return nil
}

// handleSDKEvent processes events from the Copilot SDK and forwards them.
// Uses safeEventSender to protect against writing to closed channels.
func (c *CopilotClient) handleSDKEvent(sdkEvent copilot.SessionEvent, events chan<- Event, responseContent *string, closeDone func(), pendingToolCalls map[string]ToolCall) {
	switch sdkEvent.Type {
	case "assistant.message_delta":
		// Streaming text chunk
		if sdkEvent.Data.DeltaContent == nil {
			return
		}

		text := *sdkEvent.Data.DeltaContent
		*responseContent += text
		_ = safeEventSender(events, NewTextEvent(text))

	case "assistant.message":
		// Complete assistant message
		if sdkEvent.Data.Content != nil {
			// Only emit if we haven't already sent the content via deltas
			if *responseContent == "" {
				*responseContent = *sdkEvent.Data.Content
				_ = safeEventSender(events, NewTextEvent(*sdkEvent.Data.Content))
			}
		}

	case "tool.execution_start":
		// Tool execution started - the SDK handles this internally
		// We just track it for logging/UI purposes and to match with completion events
		if sdkEvent.Data.ToolName == nil {
			return
		}

		toolCall := ToolCall{
			Name: *sdkEvent.Data.ToolName,
		}

		if sdkEvent.Data.ToolCallID != nil {
			toolCall.ID = *sdkEvent.Data.ToolCallID
			// Store for matching with completion event
			pendingToolCalls[toolCall.ID] = toolCall
		}

		// Type assert Arguments to map[string]interface{} if possible
		if args, ok := sdkEvent.Data.Arguments.(map[string]any); ok {
			toolCall.Parameters = args
		}

		_ = safeEventSender(events, NewToolCallEvent(toolCall))

	case "tool.execution_complete":
		// Tool execution completed - emit result event with actual result from SDK
		var toolCall ToolCall
		if sdkEvent.Data.ToolCallID != nil {
			if tc, ok := pendingToolCalls[*sdkEvent.Data.ToolCallID]; ok {
				toolCall = tc
				delete(pendingToolCalls, *sdkEvent.Data.ToolCallID)
			}
		}

		if toolCall.Name == "" && sdkEvent.Data.ToolName != nil {
			toolCall.Name = *sdkEvent.Data.ToolName
		}

		var result string
		var toolErr error

		if sdkEvent.Data.Result != nil {
			result = sdkEvent.Data.Result.Content
		}

		if sdkEvent.Data.Success != nil && !*sdkEvent.Data.Success {
			if sdkEvent.Data.Error != nil {
				// ErrorUnion can be either ErrorClass or String
				if sdkEvent.Data.Error.ErrorClass != nil {
					toolErr = fmt.Errorf("%s", sdkEvent.Data.Error.ErrorClass.Message)
				} else if sdkEvent.Data.Error.String != nil {
					toolErr = fmt.Errorf("%s", *sdkEvent.Data.Error.String)
				}
			}
		}

		_ = safeEventSender(events, NewToolResultEvent(toolCall, result, toolErr))

	case "session.idle":
		// Session has finished processing
		closeDone()

	case "session.error":
		// Error event
		if sdkEvent.Data.Message == nil {
			return
		}

		_ = safeEventSender(events, NewErrorEvent(fmt.Errorf("SDK error: %s", *sdkEvent.Data.Message)))
	}
}
