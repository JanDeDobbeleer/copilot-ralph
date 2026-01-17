package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JanDeDobbeleer/copilot-ralph/internal/core"
)

func TestResolvePromptFromStdin(t *testing.T) {
	// Create a temp file to simulate piped stdin
	tmp := t.TempDir()
	path := filepath.Join(tmp, "prompt.txt")
	require.NoError(t, os.WriteFile(path, []byte("stdin prompt"), 0644))

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	oldStdin := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = oldStdin }()

	res, err := resolvePrompt([]string{})
	require.NoError(t, err)
	assert.Equal(t, "stdin prompt", res)
}

func TestDisplayEventsAndPrints(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	events := make(chan any, 20)
	cfg := &core.LoopConfig{MaxIterations: 5, PromisePhrase: "Done!"}

	// Send a variety of events
	go func() {
		defer close(events)
		events <- &core.LoopStartEvent{Config: cfg}
		events <- &core.IterationStartEvent{Iteration: 1, MaxIterations: 5}
		events <- &core.AIResponseEvent{Text: "Hello "}
		events <- &core.AIResponseEvent{Text: "world"}
		events <- &core.ToolExecutionStartEvent{ToolEvent: core.ToolEvent{ToolName: "echo", Iteration: 1}}
		events <- &core.ToolExecutionEvent{ToolEvent: core.ToolEvent{ToolName: "echo", Iteration: 1}, Result: "ok"}
		events <- &core.ToolExecutionEvent{ToolEvent: core.ToolEvent{ToolName: "fail", Iteration: 1}, Error: assert.AnError}
		events <- &core.IterationCompleteEvent{Iteration: 1, Duration: time.Millisecond}
		events <- &core.PromiseDetectedEvent{Phrase: "Done!"}
		// Send cancelled to stop displayEvents
		events <- &core.LoopCancelledEvent{}
	}()

	// Call displayEvents which should process until cancel
	displayEvents(events, cfg)

	// Restore stdout and read
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Basic assertions that branches ran
	assert.Contains(t, output, "Loop started")
	assert.Contains(t, output, "Iteration 1/5")
	assert.Contains(t, output, "Hello world")
	assert.Contains(t, output, "Promise detected")
}

func TestPrintLoopConfigAndSummary(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	cfg := &core.LoopConfig{Prompt: "task", Model: "gpt-4", MaxIterations: 2, Timeout: 5 * time.Minute, PromisePhrase: "Done!", WorkingDir: "."}
	printLoopConfig(cfg)

	result := &core.LoopResult{State: core.StateComplete, Iterations: 2}
	start := time.Now().Add(-2 * time.Second)
	printSummary(result, start, cfg.PromisePhrase)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	assert.Contains(t, out, "Starting Ralph Loop")
	assert.Contains(t, out, "Loop Summary")
	assert.Contains(t, out, "Iterations:")
}

func TestCreateSDKClientReturnsClient(t *testing.T) {
	// Save/restore globals that createSDKClient reads
	oldRunModel := runModel
	oldRunStreaming := runStreaming
	oldRunLogLevel := runLogLevel
	oldRunSystemMessage := runSystemMessage
	oldRunSystemMessageMode := runSystemMessageMode
	defer func() {
		runModel = oldRunModel
		runStreaming = oldRunStreaming
		runLogLevel = oldRunLogLevel
		runSystemMessage = oldRunSystemMessage
		runSystemMessageMode = oldRunSystemMessageMode
	}()

	runModel = "gpt-test"
	runStreaming = true
	runLogLevel = "info"
	runSystemMessage = ""
	runSystemMessageMode = "append"

	cfg := &core.LoopConfig{Prompt: "task", PromisePhrase: "I'm special!", Model: "gpt-test", Timeout: 30 * time.Second, MaxIterations: 1}
	client, err := createSDKClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	// Client Model should match
	assert.Equal(t, "gpt-test", client.Model())
}
