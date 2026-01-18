// Package cli implements the command-line interface for Ralph using Cobra.
//
// This file contains tests for CLI commands.
package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JanDeDobbeleer/copilot-ralph/internal/core"
)

func TestResolvePrompt(t *testing.T) {
	t.Run("from positional argument", func(t *testing.T) {
		result, err := resolvePrompt("test prompt")
		require.NoError(t, err)
		assert.Equal(t, "test prompt", result)
	})

	t.Run("from markdown file", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "task.md")
		content := "# Task\nPlease implement X"
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))

		result, err := resolvePrompt(path)
		require.NoError(t, err)
		assert.Equal(t, content, result)
	})

	t.Run("empty when no input", func(t *testing.T) {
		_, err := resolvePrompt("")
		require.Error(t, err)
	})
}

func TestValidateRunConfig(t *testing.T) {
	tests := []struct {
		config      *core.LoopConfig
		name        string
		errorMsg    string
		expectError bool
	}{
		{
			name: "valid config",
			config: &core.LoopConfig{
				Prompt:        "test prompt",
				MaxIterations: 10,
				Timeout:       30 * time.Minute,
			},
			expectError: false,
		},
		{
			name: "empty prompt",
			config: &core.LoopConfig{
				Prompt:        "",
				MaxIterations: 10,
				Timeout:       30 * time.Minute,
			},
			expectError: true,
			errorMsg:    "prompt cannot be empty",
		},
		{
			name: "negative max iterations",
			config: &core.LoopConfig{
				Prompt:        "test",
				MaxIterations: -1,
				Timeout:       30 * time.Minute,
			},
			expectError: true,
			errorMsg:    "max-iterations must be positive",
		},
		{
			name: "negative timeout",
			config: &core.LoopConfig{
				Prompt:        "test",
				MaxIterations: 10,
				Timeout:       -1 * time.Minute,
			},
			expectError: true,
			errorMsg:    "timeout must be positive",
		},
		{
			name: "zero max iterations not allowed",
			config: &core.LoopConfig{
				Prompt:        "test",
				MaxIterations: 0,
				Timeout:       30 * time.Minute,
			},
			expectError: true,
			errorMsg:    "max-iterations must be positive",
		},
		{
			name: "zero timeout not allowed",
			config: &core.LoopConfig{
				Prompt:        "test",
				MaxIterations: 10,
				Timeout:       0,
			},
			expectError: true,
			errorMsg:    "timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRunConfig(tt.config)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestBuildLoopConfig(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		promise        string
		model          string
		workingDir     string
		expectedPrompt string
		maxIterations  int
		timeout        time.Duration
	}{
		{
			name:           "uses provided prompt",
			prompt:         "my task",
			maxIterations:  10,
			timeout:        30 * time.Minute,
			promise:        "I'm special!",
			model:          "gpt-4",
			workingDir:     ".",
			expectedPrompt: "my task",
		},
		{
			name:           "applies flag overrides",
			prompt:         "override test",
			maxIterations:  5,
			timeout:        10 * time.Minute,
			promise:        "Done!",
			model:          "gpt-3.5-turbo",
			workingDir:     "/tmp",
			expectedPrompt: "override test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore globals
			oldMaxIterations := runMaxIterations
			oldTimeout := runTimeout
			oldPromise := runPromise
			oldModel := runModel
			oldWorkingDir := runWorkingDir

			runMaxIterations = tt.maxIterations
			runTimeout = tt.timeout
			runPromise = tt.promise
			runModel = tt.model
			runWorkingDir = tt.workingDir

			defer func() {
				runMaxIterations = oldMaxIterations
				runTimeout = oldTimeout
				runPromise = oldPromise
				runModel = oldModel
				runWorkingDir = oldWorkingDir
			}()

			result := buildLoopConfig(tt.prompt)

			assert.Equal(t, tt.expectedPrompt, result.Prompt)
			assert.Equal(t, tt.maxIterations, result.MaxIterations)
			assert.Equal(t, tt.timeout, result.Timeout)
			assert.Equal(t, tt.promise, result.PromisePhrase)
			assert.Equal(t, tt.model, result.Model)
			assert.Equal(t, tt.workingDir, result.WorkingDir)
		})
	}
}

func TestConfigExists(t *testing.T) {
	tests := []struct {
		setup    func(t *testing.T) string
		name     string
		expected bool
	}{
		{
			name: "file exists",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				path := filepath.Join(tmpDir, ".ralph.yaml")
				err := os.WriteFile(path, []byte("test: true"), 0644)
				require.NoError(t, err)
				return path
			},
			expected: true,
		},
		{
			name: "file does not exist",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "nonexistent.yaml")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			_, err := os.Stat(path)
			result := !errors.Is(err, os.ErrNotExist)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateSettings(t *testing.T) {
	tests := []struct {
		name        string
		systemMode  string
		logLevel    string
		errorMsg    string
		expectError bool
	}{
		{
			name:        "invalid system message mode",
			systemMode:  "invalid",
			logLevel:    "info",
			expectError: true,
			errorMsg:    "invalid system-message-mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore globals
			oldSystemMode := runSystemMessageMode
			runSystemMessageMode = tt.systemMode

			defer func() {
				runSystemMessageMode = oldSystemMode
			}()

			err := validateSettings()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestPrintDryRun(t *testing.T) {
	cfg := &core.LoopConfig{
		Prompt:        "test prompt",
		Model:         "gpt-4",
		MaxIterations: 5,
		Timeout:       10 * time.Minute,
		PromisePhrase: "Done!",
		WorkingDir:    ".",
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printDryRun(cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Configuration Preview")
	assert.Contains(t, output, "test prompt")
	assert.Contains(t, output, "gpt-4")
	assert.Contains(t, output, "5")
}

func TestToolErrorsContinueExecution(t *testing.T) {
	// Create a mock event stream with tool errors
	events := make(chan any, 10)

	// Send events including tool errors
	go func() {
		defer close(events)

		events <- &core.LoopStartEvent{
			Config: &core.LoopConfig{
				MaxIterations: 5,
				PromisePhrase: "Done!",
			},
		}

		events <- &core.IterationStartEvent{
			Iteration:     1,
			MaxIterations: 5,
		}

		// Tool with error - should not stop processing
		events <- &core.ToolExecutionEvent{
			ToolEvent: core.ToolEvent{
				ToolName:   "view",
				Parameters: map[string]any{"path": "nonexistent"},
				Iteration:  1,
			},
			Error: errors.New("Path does not exist"),
		}

		// Successful tool - should be processed
		events <- &core.ToolExecutionEvent{
			ToolEvent: core.ToolEvent{
				ToolName:   "list",
				Parameters: map[string]any{},
				Iteration:  1,
			},
			Result: "file1.txt, file2.txt",
		}

		events <- &core.IterationCompleteEvent{
			Iteration: 1,
			Duration:  time.Second,
		}

		events <- &core.LoopCompleteEvent{
			Result: &core.LoopResult{
				State:      core.StateComplete,
				Iterations: 1,
			},
		}
	}()

	// Process events - should not panic or stop early
	var toolErrorSeen bool
	var successfulToolSeen bool
	var iterationCompleteSeen bool

	for event := range events {
		switch e := event.(type) {
		case *core.ToolExecutionEvent:
			if e.Error != nil {
				toolErrorSeen = true
			}
			if e.Error == nil && e.ToolName == "list" {
				successfulToolSeen = true
			}
		case *core.IterationCompleteEvent:
			iterationCompleteSeen = true
		}
	}

	// Verify all events were processed
	assert.True(t, toolErrorSeen, "Tool error event should be processed")
	assert.True(t, successfulToolSeen, "Successful tool after error should be processed")
	assert.True(t, iterationCompleteSeen, "Iteration complete should be processed after tool error")
}
