// Package cli implements the command-line interface for Ralph using Cobra.
//
// This file implements the `ralph run` command for executing AI development loops.
//
// See specs/cli.md for detailed CLI specification.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/JanDeDobbeleer/copilot-ralph/internal/core"
	"github.com/JanDeDobbeleer/copilot-ralph/internal/sdk"
	"github.com/JanDeDobbeleer/copilot-ralph/internal/tui/styles"
)

// Exit codes per spec
const (
	exitSuccess       = 0
	exitFailed        = 1
	exitCancelled     = 2
	exitTimeout       = 3
	exitMaxIterations = 4
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run <prompt>",
	Short: "Run an AI development loop",
	Long: `Run an AI development loop with a prompt.

The loop will iterate until the AI outputs the completion promise phrase,
reaches the maximum iterations, or times out.

Examples:
  # Direct prompt (required)
  ralph run "Add unit tests for the parser module"

  # Using a Markdown file as prompt
  ralph run task_description.md

  # With options
  ralph run --max-iterations 5 --timeout 10m "Refactor authentication"

  # From stdin
  echo "Fix linting errors" | ralph run

  # Dry run
  ralph run --dry-run "Update documentation"

  # Override promise phrase
  ralph run --promise "Task complete!" "Fix bug"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLoop,
}

var (
	runMaxIterations    int
	runTimeout          time.Duration
	runPromise          string
	runModel            string
	runWorkingDir       string
	runDryRun           bool
	runStreaming        bool
	runSystemPrompt     string
	runSystemPromptMode string
	runLogLevel         string
)

func init() {
	runCmd.Flags().IntVarP(&runMaxIterations, "max-iterations", "m", 10, "maximum loop iterations")
	runCmd.Flags().DurationVarP(&runTimeout, "timeout", "t", 30*time.Minute, "maximum loop runtime")
	runCmd.Flags().StringVar(&runPromise, "promise", "I'm special!", "completion promise phrase")
	runCmd.Flags().StringVar(&runModel, "model", "gpt-4", "AI model to use")
	runCmd.Flags().StringVar(&runWorkingDir, "working-dir", ".", "working directory for loop execution")
	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false, "show what would be executed without running")
	runCmd.Flags().BoolVar(&runStreaming, "streaming", true, "enable streaming responses")
	runCmd.Flags().StringVar(&runSystemPrompt, "system-prompt", "", "custom system message, can be a prompt or path to Markdown file")
	runCmd.Flags().StringVar(&runSystemPromptMode, "system-prompt-mode", "append", "system message mode: append or replace")
	runCmd.Flags().StringVar(&runLogLevel, "log-level", "info", "log level: debug, info, warn, error")
}

// runLoop executes the AI development loop.
func runLoop(cmd *cobra.Command, args []string) error {
	// Resolve prompt from arguments, flag, or stdin
	prompt, err := resolvePrompt(args[0])
	if err != nil {
		return err
	}

	// Require prompt
	if prompt == "" {
		return errors.New("prompt is required (provide as argument or via stdin)")
	}

	// Build loop configuration from flags
	loopConfig := buildLoopConfig(prompt)

	// Validate configuration
	if err := validateRunConfig(loopConfig); err != nil {
		return err
	}

	// Validate additional settings
	if err := validateSettings(); err != nil {
		return err
	}

	// Handle dry run
	if loopConfig.DryRun {
		return printDryRun(loopConfig)
	}

	// Print configuration
	printLoopConfig(loopConfig)

	// Create SDK client
	sdkClient, err := createSDKClient(loopConfig)
	if err != nil {
		return fmt.Errorf("failed to create SDK client: %w", err)
	}
	defer sdkClient.Stop()

	// Start SDK client
	if err := sdkClient.Start(); err != nil {
		return fmt.Errorf("failed to start SDK client: %w", err)
	}

	// Create loop engine
	engine := core.NewLoopEngine(loopConfig, sdkClient)

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start event listener to display progress
	startTime := time.Now()
	eventsDone := make(chan struct{})
	go func() {
		displayEvents(engine.Events(), loopConfig)
		close(eventsDone)
	}()

	// Start the loop in a goroutine
	resultCh := make(chan *core.LoopResult, 1)
	go func() {
		result, _ := engine.Start(ctx)
		// Always send the result, even if there's an error
		// (engine returns both result and error for cancellation/failure)
		resultCh <- result
	}()

	// Wait for completion or interrupt
	var result *core.LoopResult

	select {
	case <-sigCh:
		fmt.Println(styles.WarningStyle.Render("\n⚠ Received interrupt signal, cancelling loop..."))
		// Stop listening for more signals immediately
		signal.Stop(sigCh)
		cancel()

		// Set up force exit on second interrupt
		go func() {
			<-sigCh
			fmt.Println(styles.ErrorStyle.Render("\n⚠ Second interrupt received, forcing exit..."))
			os.Exit(exitCancelled)
		}()
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		// Wait for the loop to finish cancelling
		result = <-resultCh
	case result = <-resultCh:
		// Normal completion or error
		signal.Stop(sigCh)
	}

	// Wait for events to finish displaying (with timeout to prevent hanging)
	select {
	case <-eventsDone:
		// Events finished normally
	case <-time.After(1 * time.Second):
		// Timeout waiting for events - continue anyway
	}

	// Stop any remaining signal handlers
	signal.Stop(sigCh)

	// Print summary if we have a result
	if result != nil {
		printSummary(result, startTime)
	}

	// Always exit with appropriate code - never return to let Cobra continue
	if result == nil {
		os.Exit(exitCancelled)
	}

	switch result.State {
	case core.StateComplete:
		os.Exit(exitSuccess)
	case core.StateCancelled:
		os.Exit(exitCancelled)
	case core.StateFailed:
		if result.Error != nil {
			if errors.Is(result.Error, context.DeadlineExceeded) || errors.Is(result.Error, core.ErrLoopTimeout) {
				os.Exit(exitTimeout)
			}
			if errors.Is(result.Error, core.ErrMaxIterations) {
				os.Exit(exitMaxIterations)
			}
		}
		os.Exit(exitFailed)
	default:
		os.Exit(exitFailed)
	}

	return nil
}

// resolvePrompt determines the prompt from various sources.
// Supports:
//   - Positional argument (direct text prompt)
//   - Positional argument as a path to a Markdown file (.md/.markdown)
//   - Stdin (if piped)
func resolvePrompt(prompt string) (string, error) {
	// Priority 1: Positional argument
	if prompt == "" {
		return "", errors.New("no prompt provided")
	}

	info, err := os.Stat(prompt)
	if err != nil {
		return prompt, nil
	}

	if info.IsDir() {
		return "", fmt.Errorf("prompt path %s is a directory, must be a Markdown file", prompt)
	}

	ext := strings.ToLower(filepath.Ext(prompt))
	if ext != ".md" && ext != ".markdown" {
		return "", fmt.Errorf("file %s must be a Markdown file with extension .md or .markdown", prompt)
	}

	data, err := os.ReadFile(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file %s: %w", prompt, err)
	}

	return string(data), nil
}

// buildLoopConfig creates a LoopConfig from command-line flags.
func buildLoopConfig(prompt string) *core.LoopConfig {
	return &core.LoopConfig{
		Prompt:        prompt,
		MaxIterations: runMaxIterations,
		Timeout:       runTimeout,
		PromisePhrase: runPromise,
		Model:         runModel,
		WorkingDir:    runWorkingDir,
		DryRun:        runDryRun,
	}
}

// validateRunConfig validates the loop configuration.
func validateRunConfig(cfg *core.LoopConfig) error {
	if cfg.Prompt == "" {
		return errors.New("prompt cannot be empty")
	}

	if cfg.MaxIterations <= 0 {
		return fmt.Errorf("max-iterations must be positive (got: %d)", cfg.MaxIterations)
	}

	if cfg.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive (got: %v)", cfg.Timeout)
	}

	return nil
}

// validateSettings validates additional CLI flag settings.
func validateSettings() error {
	// Validate system message mode
	if runSystemPromptMode != "append" && runSystemPromptMode != "replace" {
		return fmt.Errorf("invalid system-prompt-mode: %q (must be append or replace)", runSystemPromptMode)
	}

	return nil
}

// printDryRun displays what would be executed without running.
func printDryRun(cfg *core.LoopConfig) error {
	fmt.Println(styles.TitleStyle.Render("🔍 Dry Run - Configuration Preview"))
	fmt.Println()
	fmt.Println(styles.InfoStyle.Render("  Prompt:            ") + cfg.Prompt)
	fmt.Println(styles.InfoStyle.Render("  Model:             ") + cfg.Model)
	fmt.Println(styles.InfoStyle.Render("  Max iterations:    ") + fmt.Sprintf("%d", cfg.MaxIterations))
	fmt.Println(styles.InfoStyle.Render("  Timeout:           ") + cfg.Timeout.String())
	fmt.Println(styles.InfoStyle.Render("  Promise phrase:    ") + cfg.PromisePhrase)
	fmt.Println(styles.InfoStyle.Render("  Working directory: ") + cfg.WorkingDir)
	fmt.Println()
	return nil
}

// printLoopConfig displays the loop configuration before starting.
func printLoopConfig(cfg *core.LoopConfig) {
	// Print Ralph ASCII art
	ralphStyle := lipgloss.NewStyle().Foreground(styles.Info)
	fmt.Println(ralphStyle.Render(styles.RalphWiggum))
	fmt.Println()

	fmt.Println(styles.TitleStyle.Render("▶  Starting Ralph Loop"))
	fmt.Println(styles.WarningStyle.Render("Prompt:         ") + cfg.Prompt)
	fmt.Println(styles.WarningStyle.Render("Model:          ") + cfg.Model)
	fmt.Println(styles.WarningStyle.Render("Max iterations: ") + fmt.Sprintf("%d", cfg.MaxIterations))
	fmt.Println(styles.WarningStyle.Render("Timeout:        ") + cfg.Timeout.String())
	fmt.Println(styles.WarningStyle.Render("Working dir:    ") + cfg.WorkingDir)
	fmt.Println()
}

// displayEvents listens for loop events and displays them to stdout.
func displayEvents(events <-chan any, cfg *core.LoopConfig) {
	var aiResponseBuffer strings.Builder
	var lastEvent any

	for event := range events {
		switch e := event.(type) {
		case *core.LoopStartEvent:
			fmt.Println(styles.PrimaryStyle.Render("▶ Loop started"))
			fmt.Println()

		case *core.IterationStartEvent:
			fmt.Println(styles.TitleStyle.Render(fmt.Sprintf("━━━ Iteration %d/%d ━━━",
				e.Iteration, cfg.MaxIterations)))

		case *core.AIResponseEvent:
			// Accumulate AI response text
			aiResponseBuffer.WriteString(e.Text)
			// Print as we receive it for streaming effect
			fmt.Print(e.Text)

		case *core.ToolExecutionStartEvent:
			// Print newline if previous event was AI response
			if _, ok := lastEvent.(*core.AIResponseEvent); ok {
				fmt.Println()
			}
			fmt.Println(styles.InfoStyle.Render(e.Info("🛠️")))

		case *core.ToolExecutionEvent:
			if e.Error != nil {
				err := styles.ErrorStyle.Render(fmt.Sprintf("(%s)", e.Error))
				fmt.Printf("%s %s\n", e.Info("❌"), err)
			} else {
				fmt.Println(styles.SuccessStyle.Render(e.Info("✔️")))
			}

		case *core.IterationCompleteEvent:
			// Print newline if previous event was AI response
			if _, ok := lastEvent.(*core.AIResponseEvent); ok {
				fmt.Println()
			}
			fmt.Println(styles.InfoStyle.Render(fmt.Sprintf("✓ Iteration %d complete", e.Iteration)))
			aiResponseBuffer.Reset() // Clear buffer for next iteration

		case *core.PromiseDetectedEvent:
			// Print newline if previous event was AI response
			if _, ok := lastEvent.(*core.AIResponseEvent); ok {
				fmt.Println()
			}
			fmt.Println(styles.SuccessStyle.Render(fmt.Sprintf("🎉 Promise detected: \"%s\"", e.Phrase)))

		case *core.ErrorEvent:
			// Print newline if previous event was AI response
			if _, ok := lastEvent.(*core.AIResponseEvent); ok {
				fmt.Println()
			}
			fmt.Println(styles.ErrorStyle.Render(fmt.Sprintf("✗ Error: %v", e.Error)))

		case *core.LoopCompleteEvent:
			// Will be handled by summary
			return

		case *core.LoopFailedEvent:
			// Will be handled by summary
			return

		case *core.LoopCancelledEvent:
			// Print newline if previous event was AI response
			if _, ok := lastEvent.(*core.AIResponseEvent); ok {
				fmt.Println()
			}
			fmt.Println(styles.WarningStyle.Render("⚠ Loop cancelled"))
			return
		}

		// Store current event as last event for next iteration
		lastEvent = event
	}
}

// printSummary displays the final loop summary.
func printSummary(result *core.LoopResult, startTime time.Time) {
	duration := time.Since(startTime)

	fmt.Println()
	fmt.Println(styles.PrimaryStyle.Render("📊 Loop Summary"))
	fmt.Println()

	// Status with color
	var status string
	switch result.State {
	case core.StateComplete:
		status = styles.SuccessStyle.Render("✓ Complete")
	case core.StateFailed:
		status = styles.ErrorStyle.Render("✗ Failed")
	case core.StateCancelled:
		status = styles.WarningStyle.Render("⚠ Cancelled")
	default:
		status = result.State.String()
	}

	fmt.Println(styles.InfoStyle.Render("Status:     ") + status)
	fmt.Println(styles.InfoStyle.Render("Iterations: ") + fmt.Sprintf("%d", result.Iterations))
	fmt.Println(styles.InfoStyle.Render("Duration:   ") + duration.Round(time.Second).String())

	if result.Error != nil {
		fmt.Println(styles.ErrorStyle.Render("Error:      ") + result.Error.Error())
	}

	fmt.Println()
}

// createSDKClient creates an SDK client with the given configuration.
func createSDKClient(loopConfig *core.LoopConfig) (*sdk.CopilotClient, error) {
	opts := []sdk.ClientOption{
		sdk.WithModel(loopConfig.Model),
		sdk.WithWorkingDir(loopConfig.WorkingDir),
		sdk.WithTimeout(loopConfig.Timeout),
		sdk.WithStreaming(runStreaming),
		sdk.WithLogLevel(runLogLevel),
	}

	// Build system prompt from template with user's task and promise phrase
	systemPrompt := core.BuildSystemPrompt(loopConfig.PromisePhrase)

	// Use the built-in system prompt, or override if user specified custom one
	if runSystemPrompt != "" {
		systemPrompt, err := resolvePrompt(runSystemPrompt)
		if err != nil {
			return nil, err
		}

		opts = append(opts, sdk.WithSystemMessage(systemPrompt, runSystemPromptMode))
	} else {
		// Use the default system prompt template with "append" mode
		opts = append(opts, sdk.WithSystemMessage(systemPrompt, "append"))
	}

	return sdk.NewCopilotClient(opts...)
}
