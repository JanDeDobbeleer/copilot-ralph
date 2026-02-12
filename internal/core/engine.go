// Package core provides the loop engine execution logic.

package core

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/JanDeDobbeleer/copilot-ralph/internal/sdk"
)

//go:embed system.md
var systemPromptTemplate string

// ErrLoopCancelled indicates the loop was cancelled by the user.
var ErrLoopCancelled = errors.New("loop cancelled")

// ErrLoopTimeout indicates the loop exceeded the configured timeout.
var ErrLoopTimeout = errors.New("loop timeout exceeded")

// ErrMaxIterations indicates the maximum iterations were reached.
var ErrMaxIterations = errors.New("maximum iterations reached")

// Start begins loop execution and runs until completion, failure, or cancellation.
// It returns the loop result containing statistics and outcome.
// The provided context can be used to cancel execution externally.
func (e *LoopEngine) Start(ctx context.Context) (*LoopResult, error) {
	e.mu.Lock()
	if e.state != StateIdle {
		e.mu.Unlock()
		return nil, errors.New("loop already running")
	}

	// Set up cancellation with timeout if configured
	if e.config.Timeout > 0 {
		e.ctx, e.cancel = context.WithTimeout(ctx, e.config.Timeout)
	} else {
		e.ctx, e.cancel = context.WithCancel(ctx)
	}
	e.state = StateRunning
	e.startTime = time.Now()
	e.iteration = 0
	e.mu.Unlock()

	// Close events channel when engine finishes to unblock any listeners
	defer func() {
		e.mu.Lock()
		e.eventsClosed = true
		e.mu.Unlock()
		close(e.events)
	}()

	// Emit loop start event
	e.emit(NewLoopStartEvent(e.config))

	// Initialize SDK if provided
	if e.sdk != nil {
		if err := e.sdk.Start(e.ctx); err != nil {
			return e.fail(fmt.Errorf("failed to start SDK: %w", err))
		}

		err := e.sdk.CreateSession(e.ctx)
		if err != nil {
			return e.fail(fmt.Errorf("failed to create SDK session: %w", err))
		}
	}

	// Run the main loop
	result, err := e.runLoop()

	// Clean up SDK - do it in background if cancelled for immediate return
	if e.sdk != nil {
		cleanupBaseCtx := context.WithoutCancel(e.ctx)

		if result != nil && result.State == StateCancelled {
			// Background cleanup on cancellation - don't wait
			go func() {
				cleanupCtx, cleanupCancel := context.WithTimeout(cleanupBaseCtx, 1*time.Second)
				defer cleanupCancel()
				_ = e.sdk.DestroySession(cleanupCtx)
				_ = e.sdk.Stop()
			}()
		} else {
			// Normal cleanup - wait for completion
			cleanupCtx, cleanupCancel := context.WithTimeout(cleanupBaseCtx, 5*time.Second)
			_ = e.sdk.DestroySession(cleanupCtx)
			cleanupCancel()
			_ = e.sdk.Stop()
		}
	}

	return result, err
}

// runLoop executes the main iteration loop.
// The loop continues until all iterations are completed, timeout is hit, or an error occurs.
// Promise detection is tracked but does not stop the loop.
func (e *LoopEngine) runLoop() (*LoopResult, error) {
	for {
		if result, err := e.preIterationCheck(); err != nil || result != nil {
			return result, err
		}

		// Execute iteration
		err := e.executeIteration()
		if err != nil {
			// Check if it's a timeout (context deadline exceeded)
			if errors.Is(err, context.DeadlineExceeded) {
				return e.fail(ErrLoopTimeout)
			}
			// Check if it's a cancellation
			if errors.Is(err, context.Canceled) {
				return e.cancelled()
			}
			return e.fail(fmt.Errorf("iteration %d failed: %w", e.iteration, err))
		}

		// Continue to next iteration (promise does not stop the loop)
	}
}

// preIterationCheck evaluates cancellation, state, and limit guards before running an iteration.
func (e *LoopEngine) preIterationCheck() (*LoopResult, error) {
	select {
	case <-e.ctx.Done():
		return e.cancelled()
	default:
	}

	e.mu.RLock()
	state := e.state
	e.mu.RUnlock()

	if state == StateCancelled {
		return e.cancelled()
	}

	if e.config.Timeout > 0 && time.Since(e.startTime) > e.config.Timeout {
		return e.fail(ErrLoopTimeout)
	}

	// Check if max iterations have been reached BEFORE starting a new one
	if e.config.MaxIterations > 0 && e.iteration >= e.config.MaxIterations {
		return e.complete()
	}

	return nil, nil
}

// executeIteration executes a single iteration of the loop.
// Promise detection is tracked but does not stop execution.
func (e *LoopEngine) executeIteration() error {
	e.mu.Lock()
	e.iteration++
	iteration := e.iteration
	e.mu.Unlock()

	iterationStart := time.Now()

	// Emit iteration start
	e.emit(NewIterationStartEvent(iteration, e.config.MaxIterations))

	// Build context and send prompt
	prompt := e.buildIterationPrompt(iteration)

	// If SDK is available, send prompt
	if e.sdk != nil {
		events, err := e.sdk.SendPrompt(e.ctx, prompt)
		if err != nil {
			return fmt.Errorf("failed to send prompt: %w", err)
		}

		// Process events - use select to handle both events and cancellation
	eventLoop:
		for {
			select {
			case <-e.ctx.Done():
				return e.ctx.Err()
			case event, ok := <-events:
				if !ok {
					// Channel closed, exit loop
					break eventLoop
				}

				switch ev := event.(type) {
				case *sdk.TextEvent:
					e.emit(NewAIResponseEvent(ev.Text, iteration))

					// Check for promise in streaming text that's not reasoning
					if !ev.Reasoning && detectPromise(ev.Text, e.config.PromisePhrase) {
						e.emit(NewPromiseDetectedEvent(e.config.PromisePhrase, "ai_response", iteration))
					}

				case *sdk.ToolCallEvent:
					// Tool execution started - SDK handles it internally
					// We just log the start for UI purposes
					e.emit(NewToolExecutionStartEvent(
						ev.ToolCall.Name,
						ev.ToolCall.Parameters,
						iteration,
					))

				case *sdk.ToolResultEvent:
					e.emit(NewToolExecutionEvent(
						ev.ToolCall.Name,
						ev.ToolCall.Parameters,
						ev.Result,
						ev.Error,
						0, // Duration not available from SDK events
						iteration,
					))

				case *sdk.ErrorEvent:
					// SDK errors are typically tool execution failures, which are recoverable
					e.emit(NewErrorEvent(ev.Err, iteration, true))
				}
			}
		}
	}

	iterationDuration := time.Since(iterationStart)

	// Emit iteration complete
	e.emit(NewIterationCompleteEvent(iteration, iterationDuration))

	return nil
}

// buildIterationPrompt builds the prompt for the current iteration.
// The system prompt template handles the loop context and completion instructions.
func (e *LoopEngine) buildIterationPrompt(iteration int) string {
	var builder strings.Builder

	// Add iteration context
	builder.WriteString(fmt.Sprintf("[Iteration %d/%d]\n\n", iteration, e.config.MaxIterations))

	// Add original task prompt
	builder.WriteString(e.config.Prompt)

	return builder.String()
}

// complete transitions to the complete state and returns the result.
// This is now only called when max iterations are reached without errors.
func (e *LoopEngine) complete() (*LoopResult, error) {
	e.mu.Lock()
	e.state = StateComplete
	result := e.buildResult()
	result.State = StateComplete
	e.mu.Unlock()

	e.emit(NewLoopCompleteEvent(result))

	return result, nil
}

// fail transitions to the failed state and returns the result with error.
func (e *LoopEngine) fail(err error) (*LoopResult, error) {
	e.mu.Lock()
	e.state = StateFailed
	result := e.buildResult()
	result.State = StateFailed
	result.Error = err
	e.mu.Unlock()

	e.emit(NewLoopFailedEvent(err, result))

	return result, err
}

// cancelled transitions to the cancelled state and returns the result.
func (e *LoopEngine) cancelled() (*LoopResult, error) {
	e.mu.Lock()
	e.state = StateCancelled
	result := e.buildResult()
	result.State = StateCancelled
	result.Error = ErrLoopCancelled
	e.mu.Unlock()

	e.emit(NewLoopCancelledEvent(result))

	return result, ErrLoopCancelled
}

// buildResult creates a LoopResult from current state.
// Must be called with lock held.
func (e *LoopEngine) buildResult() *LoopResult {
	return &LoopResult{
		State:      e.state,
		Iterations: e.iteration,
		Duration:   time.Since(e.startTime),
	}
}

// emit sends an event to the events channel.
func (e *LoopEngine) emit(event any) {
	e.mu.RLock()
	closed := e.eventsClosed
	e.mu.RUnlock()

	if closed {
		return
	}

	select {
	case e.events <- event:
	default:
		// Channel full, event dropped - log warning
	}
}
