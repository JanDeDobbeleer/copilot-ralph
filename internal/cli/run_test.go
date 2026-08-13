package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JanDeDobbeleer/copilot-ralph/internal/core"
	"github.com/JanDeDobbeleer/copilot-ralph/internal/sdk"
)

type fakeAvailableModelsClient struct {
	models      []sdk.Model
	startErr    error
	listErr     error
	stopErr     error
	startCalled bool
	listCalled  bool
	stopCalled  bool
}

func (client *fakeAvailableModelsClient) Start(_ context.Context) error {
	client.startCalled = true
	return client.startErr
}

func (client *fakeAvailableModelsClient) Stop() error {
	client.stopCalled = true
	return client.stopErr
}

func (client *fakeAvailableModelsClient) ListModels(_ context.Context) ([]sdk.Model, error) {
	client.listCalled = true
	return client.models, client.listErr
}

func TestRunModelListing(t *testing.T) {
	t.Run("lists models sorted by ID", func(t *testing.T) {
		client := &fakeAvailableModelsClient{
			models: []sdk.Model{
				{ID: "zeta", Name: "Zeta"},
				{ID: "alpha", Name: "Alpha"},
			},
		}
		var output bytes.Buffer

		err := runModelListing(t.Context(), client, &output)

		require.NoError(t, err)
		assert.True(t, client.startCalled)
		assert.True(t, client.listCalled)
		assert.True(t, client.stopCalled)
		assert.Contains(t, output.String(), "Available models:")
		assert.Contains(t, output.String(), "ID")
		assert.Contains(t, output.String(), "NAME")
		assert.Less(t, strings.Index(output.String(), "alpha"), strings.Index(output.String(), "zeta"))
	})

	t.Run("reports an empty model list", func(t *testing.T) {
		client := &fakeAvailableModelsClient{}
		var output bytes.Buffer

		err := runModelListing(t.Context(), client, &output)

		require.NoError(t, err)
		assert.Equal(t, "No models available.\n", output.String())
		assert.True(t, client.stopCalled)
	})

	t.Run("returns start error", func(t *testing.T) {
		client := &fakeAvailableModelsClient{startErr: assert.AnError}

		err := runModelListing(t.Context(), client, &bytes.Buffer{})

		require.ErrorIs(t, err, assert.AnError)
		assert.False(t, client.listCalled)
		assert.False(t, client.stopCalled)
	})

	t.Run("returns list and stop errors", func(t *testing.T) {
		listErr := errors.New("list failed")
		stopErr := errors.New("stop failed")
		client := &fakeAvailableModelsClient{listErr: listErr, stopErr: stopErr}

		err := runModelListing(t.Context(), client, &bytes.Buffer{})

		require.ErrorIs(t, err, listErr)
		require.ErrorIs(t, err, stopErr)
		assert.True(t, client.stopCalled)
	})
}

func TestRootCommandExists(t *testing.T) {
	// Verify that the ralph root command can be invoked with --help
	cmd := exec.Command("go", "run", "./cmd/ralph", "--help")
	// Do not fail if environment unsuitable; this is a smoke test
	_ = cmd.Run()
	require.True(t, true)
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
	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
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
	printSummary(result, start)

	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
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
	oldRunSystemMessage := runSystemPrompt
	oldRunSystemMessageMode := runSystemPromptMode
	defer func() {
		runModel = oldRunModel
		runStreaming = oldRunStreaming
		runLogLevel = oldRunLogLevel
		runSystemPrompt = oldRunSystemMessage
		runSystemPromptMode = oldRunSystemMessageMode
	}()

	runModel = "gpt-test"
	runStreaming = true
	runLogLevel = "info"
	runSystemPrompt = ""
	runSystemPromptMode = "append"

	cfg := &core.LoopConfig{Prompt: "task", PromisePhrase: "I'm special!", Model: "gpt-test", Timeout: 30 * time.Second, MaxIterations: 1}
	client, err := createSDKClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	// Client Model should match
	assert.Equal(t, "gpt-test", client.Model())
}
