// Package core provides loop event types for the loop engine.

package core

import (
	"fmt"
	"strings"
	"time"
)

// LoopStartEvent indicates the loop has started.
type LoopStartEvent struct {
	// Config is the loop configuration.
	Config *LoopConfig
}

// NewLoopStartEvent creates a new LoopStartEvent.
func NewLoopStartEvent(config *LoopConfig) *LoopStartEvent {
	return &LoopStartEvent{
		Config: config,
	}
}

// LoopCompleteEvent indicates the loop completed successfully.
type LoopCompleteEvent struct {
	// Result contains the loop result.
	Result *LoopResult
}

// NewLoopCompleteEvent creates a new LoopCompleteEvent.
func NewLoopCompleteEvent(result *LoopResult) *LoopCompleteEvent {
	return &LoopCompleteEvent{
		Result: result,
	}
}

// LoopFailedEvent indicates the loop failed.
type LoopFailedEvent struct {
	// Error is the error that caused the failure.
	Error error
	// Result contains partial loop result.
	Result *LoopResult
}

// NewLoopFailedEvent creates a new LoopFailedEvent.
func NewLoopFailedEvent(err error, result *LoopResult) *LoopFailedEvent {
	return &LoopFailedEvent{
		Error:  err,
		Result: result,
	}
}

// LoopCancelledEvent indicates the loop was cancelled by the user.
type LoopCancelledEvent struct {
	// Result contains partial loop result.
	Result *LoopResult
}

// NewLoopCancelledEvent creates a new LoopCancelledEvent.
func NewLoopCancelledEvent(result *LoopResult) *LoopCancelledEvent {
	return &LoopCancelledEvent{
		Result: result,
	}
}

// IterationStartEvent indicates an iteration has started.
type IterationStartEvent struct {
	// Iteration is the iteration number (1-based).
	Iteration int
	// MaxIterations is the maximum number of iterations.
	MaxIterations int
}

// NewIterationStartEvent creates a new IterationStartEvent.
func NewIterationStartEvent(iteration, maxIterations int) *IterationStartEvent {
	return &IterationStartEvent{
		Iteration:     iteration,
		MaxIterations: maxIterations,
	}
}

// IterationCompleteEvent indicates an iteration completed.
type IterationCompleteEvent struct {
	// Iteration is the iteration number (1-based).
	Iteration int
	// Duration is how long the iteration took.
	Duration time.Duration
	// ToolsUsed is the number of tools executed.
	ToolsUsed int
	// PromiseFound indicates if the promise was found.
	PromiseFound bool
}

// NewIterationCompleteEvent creates a new IterationCompleteEvent.
func NewIterationCompleteEvent(iteration int, duration time.Duration, toolsUsed int, promiseFound bool) *IterationCompleteEvent {
	return &IterationCompleteEvent{
		Iteration:    iteration,
		Duration:     duration,
		ToolsUsed:    toolsUsed,
		PromiseFound: promiseFound,
	}
}

// AIResponseEvent indicates AI response text was received.
type AIResponseEvent struct {
	// Text is the AI response text.
	Text string
	// Iteration is the current iteration number.
	Iteration int
}

// NewAIResponseEvent creates a new AIResponseEvent.
func NewAIResponseEvent(text string, iteration int) *AIResponseEvent {
	return &AIResponseEvent{
		Text:      text,
		Iteration: iteration,
	}
}

type ToolEvent struct {
	// ToolName is the name of the tool executed.
	ToolName string
	// Parameters are the tool parameters.
	Parameters map[string]interface{}
	// Iteration is the current iteration number.
	Iteration int
}

// Info returns a formatted string describing the tool execution based on parameters.
// This provides human-readable information about what the tool is doing.
func (e *ToolEvent) Info(emoji string) string {
	if len(e.Parameters) == 0 {
		return fmt.Sprintf("%s %s", emoji, e.ToolName)
	}

	var values []string
	for _, v := range e.Parameters {
		values = append(values, fmt.Sprintf("%v", v))
	}

	return fmt.Sprintf("%s %s: %s", emoji, e.ToolName, strings.Join(values, ", "))
}

// ToolExecutionEvent indicates a tool was executed.
type ToolExecutionEvent struct {
	ToolEvent

	// Result is the tool execution result.
	Result string
	// Error is any error that occurred during execution.
	Error error
	// Duration is how long the tool took to execute.
	Duration time.Duration
}

// NewToolExecutionEvent creates a new ToolExecutionEvent.
func NewToolExecutionEvent(toolName string, params map[string]interface{}, result string, err error, duration time.Duration, iteration int) *ToolExecutionEvent {
	return &ToolExecutionEvent{
		ToolEvent: ToolEvent{
			ToolName:   toolName,
			Parameters: params,
			Iteration:  iteration,
		},
		Result:   result,
		Error:    err,
		Duration: duration,
	}
}

// ToolExecutionStartEvent indicates a tool execution has started.
type ToolExecutionStartEvent struct {
	ToolEvent
}

// NewToolExecutionStartEvent creates a new ToolExecutionStartEvent.
func NewToolExecutionStartEvent(toolName string, params map[string]interface{}, iteration int) *ToolExecutionStartEvent {
	return &ToolExecutionStartEvent{
		ToolEvent: ToolEvent{
			ToolName:   toolName,
			Parameters: params,
			Iteration:  iteration,
		},
	}
}

// PromiseDetectedEvent indicates the promise phrase was found.
type PromiseDetectedEvent struct {
	// Phrase is the promise phrase that was detected.
	Phrase string
	// Source is where the promise was found (e.g., "ai_response", "tool_output").
	Source string
	// Iteration is the iteration number where promise was found.
	Iteration int
}

// NewPromiseDetectedEvent creates a new PromiseDetectedEvent.
func NewPromiseDetectedEvent(phrase, source string, iteration int) *PromiseDetectedEvent {
	return &PromiseDetectedEvent{
		Phrase:    phrase,
		Source:    source,
		Iteration: iteration,
	}
}

// ErrorEvent indicates an error occurred.
type ErrorEvent struct {
	// Error is the error that occurred.
	Error error
	// Iteration is the current iteration number (0 if not in iteration).
	Iteration int
	// Recoverable indicates if the error is recoverable.
	Recoverable bool
}

// NewErrorEvent creates a new ErrorEvent.
func NewErrorEvent(err error, iteration int, recoverable bool) *ErrorEvent {
	return &ErrorEvent{
		Error:       err,
		Iteration:   iteration,
		Recoverable: recoverable,
	}
}
