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
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	copilot "github.com/github/copilot-sdk/go"
)

// Default configuration values.
const (
	DefaultModel     = "auto"
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
	streaming         bool
	started           bool
}

// Model describes an AI model available to the authenticated Copilot user.
type Model struct {
	ID   string
	Name string
}

type sdkClientStopper interface {
	Stop() error
	ForceStop()
}

func stopSDKClient(client sdkClientStopper) error {
	if runtime.GOOS == "windows" {
		client.ForceStop()
		return nil
	}

	return client.Stop()
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

// Start starts the client (must be called with lock held).
func (c *CopilotClient) Start(ctx context.Context) error {
	if c.started {
		return nil
	}

	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	// Initialize the SDK client with options
	c.sdkClient = copilot.NewClient(&copilot.ClientOptions{
		LogLevel:         c.logLevel,
		WorkingDirectory: c.workingDir,
	})

	// Start the SDK client
	if err := c.sdkClient.Start(ctx); err != nil {
		return fmt.Errorf("failed to start SDK client: %w", err)
	}

	c.started = true
	return nil
}

// Stop stops the client and releases resources.
func (c *CopilotClient) Stop() error {
	if !c.started {
		return nil
	}

	var stopErr error

	// Destroy any active SDK session
	if c.sdkSession != nil {
		if err := c.sdkSession.Disconnect(); err != nil {
			stopErr = errors.Join(stopErr, fmt.Errorf("failed to disconnect SDK session: %w", err))
		}
		c.sdkSession = nil
	}

	// Stop the SDK client
	if c.sdkClient != nil {
		if err := stopSDKClient(c.sdkClient); err != nil {
			stopErr = errors.Join(stopErr, fmt.Errorf("failed to stop SDK client: %w", err))
		}
		c.sdkClient = nil
	}

	c.started = false
	return stopErr
}

// CreateSession creates a new Copilot session.
// It initializes the SDK session resources and registers them with the client.
func (c *CopilotClient) CreateSession(ctx context.Context) error {
	if c.sdkClient == nil {
		return fmt.Errorf("SDK client not initialized")
	}

	// Build session config for the SDK
	sessionConfig := &copilot.SessionConfig{
		Model:               c.model,
		Streaming:           new(c.streaming),
		OnPermissionRequest: copilot.PermissionHandler.ApproveAll,
	}

	// Configure system message if provided
	if c.systemMessage != "" {
		sessionConfig.SystemMessage = &copilot.SystemMessageConfig{
			Mode:    c.systemMessageMode,
			Content: c.systemMessage,
		}
	}

	// Create SDK session
	sdkSession, err := c.sdkClient.CreateSession(ctx, sessionConfig)
	if err != nil {
		return fmt.Errorf("failed to create SDK session: %w", err)
	}

	// Store SDK session reference; we no longer maintain a local Session wrapper
	c.sdkSession = sdkSession
	return nil
}

// DestroySession destroys the current session and cleans up resources.
func (c *CopilotClient) DestroySession(ctx context.Context) error {
	if c.sdkSession == nil {
		return nil
	}

	if err := c.sdkSession.Disconnect(); err != nil {
		return fmt.Errorf("failed to disconnect SDK session: %w", err)
	}

	c.sdkSession = nil
	return nil
}

// Model returns the configured model name.
func (c *CopilotClient) Model() string {
	return c.model
}

// ListModels returns the AI models available to the authenticated Copilot user.
func (c *CopilotClient) ListModels(ctx context.Context) ([]Model, error) {
	if c.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	sdkModels, err := c.sdkClient.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	models := make([]Model, len(sdkModels))
	for index, model := range sdkModels {
		models[index] = Model{
			ID:   model.ID,
			Name: model.Name,
		}
	}

	return models, nil
}

// SendPrompt sends a prompt to the Copilot SDK and returns an event stream.
// The returned channel will be closed when the response is complete.
// An error is returned if there is no active session.
// This method includes automatic retry logic for transient errors.
func (c *CopilotClient) SendPrompt(ctx context.Context, prompt string) (<-chan Event, error) {
	if c.sdkSession == nil {
		return nil, fmt.Errorf("no active session")
	}

	// Create event channel with buffer
	events := make(chan Event, 100)

	// Process prompt asynchronously with retry logic
	go func() {
		defer close(events)
		c.sendPromptWithRetry(ctx, prompt, events)
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
func (c *CopilotClient) sendPromptWithRetry(ctx context.Context, prompt string, events chan<- Event) {
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
		err := c.sendPromptOnce(ctx, prompt, events)
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
func (c *CopilotClient) sendPromptOnce(ctx context.Context, prompt string, events chan<- Event) error {
	// Set up done channel to wait for session.idle
	done := make(chan struct{})
	doneOnce := &sync.Once{}
	closeDone := func() {
		doneOnce.Do(func() {
			close(done)
		})
	}

	var sessionErr error
	pendingToolCalls := make(map[string]ToolCall)

	// Subscribe to SDK session events
	unsubscribe := c.sdkSession.On(func(event copilot.SessionEvent) {
		// Check if context is cancelled before processing events
		select {
		case <-ctx.Done():
			// Context cancelled, close done channel to unblock and stop processing
			closeDone()
			return
		default:
		}

		if event.Type() == copilot.SessionEventTypeSessionError {
			if data, ok := event.Data.(*copilot.SessionErrorData); ok {
				sessionErr = fmt.Errorf("SDK error: %s", data.Message)
			}
		}

		c.handleSDKEvent(event, events, closeDone, pendingToolCalls)
	})

	defer unsubscribe()

	// Send the message
	_, err := c.sdkSession.Send(ctx, copilot.MessageOptions{
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
			_ = c.sdkSession.Abort(ctx)
		}()

		closeDone()
		return ctx.Err()
	case <-done:
		// Response complete - check for session error
		if sessionErr != nil {
			return sessionErr
		}
	}

	return nil
}

// handleSDKEvent processes events from the Copilot SDK and forwards them.
// Uses safeEventSender to protect against writing to closed channels.
func (c *CopilotClient) handleSDKEvent(sdkEvent copilot.SessionEvent, events chan<- Event, closeDone func(), pendingToolCalls map[string]ToolCall) {
	switch sdkEvent.Type() {
	case copilot.SessionEventTypeAssistantMessageDelta:
		data, ok := sdkEvent.Data.(*copilot.AssistantMessageDeltaData)
		if !ok {
			return
		}

		_ = safeEventSender(events, NewTextEvent(data.DeltaContent, false))

	case copilot.SessionEventTypeAssistantReasoningDelta:
		data, ok := sdkEvent.Data.(*copilot.AssistantReasoningDeltaData)
		if !ok {
			return
		}

		_ = safeEventSender(events, NewTextEvent(data.DeltaContent, true))

	case copilot.SessionEventTypeAssistantMessage:
		data, ok := sdkEvent.Data.(*copilot.AssistantMessageData)
		if !ok {
			return
		}

		_ = safeEventSender(events, NewTextEvent(data.Content, false))

	case copilot.SessionEventTypeAssistantReasoning:
		data, ok := sdkEvent.Data.(*copilot.AssistantReasoningData)
		if !ok {
			return
		}

		_ = safeEventSender(events, NewTextEvent(data.Content, true))

	case copilot.SessionEventTypeToolExecutionStart:
		data, ok := sdkEvent.Data.(*copilot.ToolExecutionStartData)
		if !ok {
			return
		}

		toolCall := ToolCall{
			ID:   data.ToolCallID,
			Name: data.ToolName,
		}

		pendingToolCalls[toolCall.ID] = toolCall

		if args, ok := data.Arguments.(map[string]any); ok {
			toolCall.Parameters = args
		}

		_ = safeEventSender(events, NewToolCallEvent(toolCall))

	case copilot.SessionEventTypeToolExecutionComplete:
		data, ok := sdkEvent.Data.(*copilot.ToolExecutionCompleteData)
		if !ok {
			return
		}

		var toolCall ToolCall
		if tc, ok := pendingToolCalls[data.ToolCallID]; ok {
			toolCall = tc
			delete(pendingToolCalls, data.ToolCallID)
		}

		var result string
		var toolErr error

		if data.Result != nil {
			result = data.Result.Content
		}

		if !data.Success && data.Error != nil {
			toolErr = fmt.Errorf("%s", data.Error.Message)
		}

		_ = safeEventSender(events, NewToolResultEvent(toolCall, result, toolErr))

	case copilot.SessionEventTypeSessionIdle:
		// Session has finished processing
		closeDone()

	case copilot.SessionEventTypeSessionError:
		data, ok := sdkEvent.Data.(*copilot.SessionErrorData)
		if !ok {
			return
		}

		_ = safeEventSender(events, NewErrorEvent(fmt.Errorf("SDK error: %s", data.Message)))
	}
}
