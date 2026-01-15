# Ralph Development Guide for AI Agents

This comprehensive guide helps AI agents understand Ralph's architecture, conventions, and development patterns. Use this as your primary reference when working on Ralph.

## Table of Contents

1. [Project Overview](#project-overview)
2. [Architecture](#architecture)
3. [Package Structure](#package-structure)
4. [Development Patterns](#development-patterns)
5. [Go Conventions](#go-conventions)
6. [Bubble Tea Patterns](#bubble-tea-patterns)
7. [Copilot SDK Integration](#copilot-sdk-integration)
8. [Testing Guidelines](#testing-guidelines)
9. [Error Handling](#error-handling)
10. [Code Style](#code-style)

## Project Overview

**Ralph** is an iterative AI development loop tool that implements the "Ralph Wiggum" technique for self-referential AI development. It continuously feeds prompts to GitHub Copilot AI, monitoring for a "completion promise" phrase, with each iteration building on previous work.

### Core Concept

```text
User provides task → Ralph starts loop → AI iterates → Makes changes → Tests →
Checks for promise → If found: complete, else: continue → Max iterations reached or timeout
```

### Technology Stack

- **Language:** Go 1.22+
- **CLI:** Cobra (command framework)
- **TUI:** Bubble Tea (terminal UI framework)
- **AI:** GitHub Copilot SDK
- **Testing:** testify (assertions and mocks)

## Architecture

### High-Level Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│                        CLI Layer                            │
│  (cmd/ralph, internal/cli)                                  │
│  - Command parsing                                          │
│  - Flag handling                                            │
│  - Entry point                                              │
└────────────────┬────────────────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────────────────┐
│                        TUI Layer                            │
│  (internal/tui)                                             │
│  - Terminal UI screens                                      │
│  - User interactions                                        │
│  - Visual feedback                                          │
└────────────────┬────────────────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────────────────┐
│                       Core Layer                            │
│  (internal/core)                                            │
│  - Loop engine                                              │
│  - State management                                         │
│  - Business logic                                           │
└────────────────┬────────────────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────────────────┐
│                       SDK Layer                             │
│  (internal/sdk)                                             │
│  - Copilot SDK wrapper                                      │
│  - Session management                                       │
│  - Tool handling                                            │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

```text
CLI Flags → Loop Config → Loop Engine → SDK Client → Copilot AI
                              ↓
                          Event Stream
                              ↓
                          TUI Update
                              ↓
                          User Display
```

## Package Structure

### `/cmd/ralph`

**Purpose:** Application entry point

**Responsibilities:**

- Parse command-line arguments
- Initialize root command
- Set up logging
- Handle version information

**Pattern:** Keep minimal, delegate to `internal/cli`

### `/internal/cli`

**Purpose:** Cobra command definitions

**Responsibilities:**

- Define commands (run, init, version)
- Register flags
- Validate inputs
- Orchestrate execution flow

**Dependencies:** `internal/tui`, `internal/core`, `pkg/version`

### `/internal/tui`

**Purpose:** Terminal user interface components

**Subdirectories:**

- `wizard/` - Initialization wizard forms
- `runner/` - Loop execution display
- `styles/` - Lip Gloss styling
- `keys/` - Key binding definitions

**Responsibilities:**

- Render UI screens
- Handle user input
- Manage screen transitions
- Display logs and status

**Dependencies:** `internal/core`, `charmbracelet/*`

**Pattern:** Model-View-Update (Bubble Tea)

### `/internal/core`

**Purpose:** Business logic and loop engine

**Responsibilities:**

- Loop state machine
- Iteration management
- Promise detection
- Event emission
- Context building

**Dependencies:** `internal/sdk`

**Pattern:** No external UI dependencies, pure logic

### `/internal/sdk`

**Purpose:** Copilot SDK integration

**Responsibilities:**

- SDK client wrapper
- Session management
- Event handling
- Tool registration
- Error handling

**Dependencies:** `github.com/github/copilot-sdk/go`

**Pattern:** Abstract SDK details from core logic

### `/pkg/version`

**Purpose:** Version information

**Responsibilities:**

- Store version, commit, build date
- Provide version strings

**Pattern:** Set via build flags (`-ldflags`)

### `/specs`

**Purpose:** Specifications (source of truth)

**Files:**

- `cli.md` - CLI interface spec
- `tui.md` - TUI behavior spec
- `loop-engine.md` - Loop engine spec
- `sdk-integration.md` - SDK integration spec

**Usage:** Reference before implementing features

## Development Patterns

### Spec-Driven Development

1. **Read the spec first** - Before coding, read relevant spec
2. **Implement per spec** - Follow spec exactly
3. **Update spec if needed** - If requirements change, update spec first
4. **Link in code** - Add comments referencing spec sections

```go
// Implements promise detection per specs/loop-engine.md#promise-detection
func (e *LoopEngine) detectPromise(text string) bool {
    // ...
}
```

### Interface-Based Design

Define interfaces at package boundaries for testability:

```go
// internal/core/loop.go
type SDKClient interface {
    SendPrompt(ctx context.Context, prompt string) (<-chan Event, error)
    RegisterTool(tool Tool) error
}

// internal/sdk/client.go implements SDKClient
```

### Dependency Injection

Pass dependencies explicitly, avoid globals:

```go
// Good
func NewLoopEngine(config *Config, sdk SDKClient) *LoopEngine {
    return &LoopEngine{
        config: config,
        sdk:    sdk,
    }
}

// Bad - global dependency
var globalSDK *CopilotClient
```

### Event-Driven Communication

Use channels for async communication:

```go
type LoopEngine struct {
    events chan LoopEvent
}

func (e *LoopEngine) Events() <-chan LoopEvent {
    return e.events
}

// Consumers
for event := range engine.Events() {
    // Handle event
}
```

## Go Conventions

### No Else Statements

**Rule:** Never use `else`. Use early returns instead.

```go
// Good
func process(data string) error {
    if data == "" {
        return errors.New("data is empty")
    }

    if err := validate(data); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    return nil
}

// Bad
func process(data string) error {
    if data == "" {
        return errors.New("data is empty")
    } else {
        if err := validate(data); err != nil {
            return fmt.Errorf("validation failed: %w", err)
        } else {
            return nil
        }
    }
}
```

### Error Wrapping

Always wrap errors with context using `%w`:

```go
// Good
if err := sdk.Connect(); err != nil {
    return fmt.Errorf("failed to connect to SDK: %w", err)
}

// Bad
if err := sdk.Connect(); err != nil {
    return err
}
```

### Error Checking

Check errors immediately, don't defer:

```go
// Good
file, err := os.Open(path)
if err != nil {
    return fmt.Errorf("failed to open file: %w", err)
}
defer file.Close()

// Bad
file, _ := os.Open(path)  // Never ignore errors
```

### Package Documentation

Every package must have a doc comment:

```go
// Package core implements the Ralph loop engine.
//
// The loop engine orchestrates iterative AI development loops,
// managing state transitions, promise detection, and event emission.
// See specs/loop-engine.md for detailed specification.
package core
```

### Exported vs Unexported

- **Exported** (capitalized): Public API
- **Unexported** (lowercase): Internal implementation

```go
// Exported - part of public API
type LoopEngine struct {
    config *LoopConfig  // unexported field
}

// Exported method
func (e *LoopEngine) Start() error {
    return e.runLoop()  // call unexported method
}

// Unexported - internal
func (e *LoopEngine) runLoop() error {
    // ...
}
```

### Table-Driven Tests

Use table-driven tests for multiple scenarios:

```go
func TestPromiseDetection(t *testing.T) {
    tests := []struct {
        name     string
        text     string
        promise  string
        expected bool
    }{
        {
            name:     "exact match",
            text:     "I'm done!",
            promise:  "I'm done!",
            expected: true,
        },
        {
            name:     "case insensitive",
            text:     "IM DONE!",
            promise:  "I'm done!",
            expected: true,
        },
        {
            name:     "not found",
            text:     "still working",
            promise:  "I'm done!",
            expected: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            engine := NewLoopEngine(&Config{PromisePhrase: tt.promise})
            result := engine.detectPromise(tt.text)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## Bubble Tea Patterns

### Model-View-Update (MVU)

All TUI components follow this pattern:

```go
type Model struct {
    // State
    state     State
    items     []string
    selected  int
    width     int
    height    int
}

// Initialize
func (m Model) Init() tea.Cmd {
    return nil  // or return initial command
}

// Update state based on messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        case "up":
            m.selected = max(0, m.selected-1)
        case "down":
            m.selected = min(len(m.items)-1, m.selected+1)
        }

    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }

    return m, nil
}

// Render to string
func (m Model) View() string {
    // Build and return display string
    return "..."
}
```

### Message Types

Define custom messages for events:

```go
type LogMsg struct {
    Level   string
    Message string
}

type IterationStartMsg struct {
    Iteration int
}

type PromiseDetectedMsg struct {
    Phrase string
}

// Send messages with commands
func checkForPromise() tea.Msg {
    // Do work
    return PromiseDetectedMsg{Phrase: "I'm done!"}
}

// In Update
case PromiseDetectedMsg:
    m.state = StateComplete
    return m, tea.Quit
```

### Commands

Commands are functions that return messages:

```go
// Simple command
func tickCmd() tea.Msg {
    time.Sleep(time.Second)
    return TickMsg{}
}

// Command with async work
func fetchDataCmd() tea.Cmd {
    return func() tea.Msg {
        data, err := fetchFromAPI()
        if err != nil {
            return ErrorMsg{err}
        }
        return DataMsg{data}
    }
}

// Batch multiple commands
return m, tea.Batch(
    tickCmd(),
    fetchDataCmd(),
    logCmd("Started"),
)
```

### Component Composition

Compose smaller components into larger ones:

```go
type RunnerModel struct {
    header   HeaderModel
    viewport viewport.Model
    spinner  spinner.Model
    footer   FooterModel
}

func (m RunnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    // Update sub-components
    var cmd tea.Cmd
    m.header, cmd = m.header.Update(msg)
    cmds = append(cmds, cmd)

    m.viewport, cmd = m.viewport.Update(msg)
    cmds = append(cmds, cmd)

    m.spinner, cmd = m.spinner.Update(msg)
    cmds = append(cmds, cmd)

    return m, tea.Batch(cmds...)
}

func (m RunnerModel) View() string {
    return lipgloss.JoinVertical(
        lipgloss.Left,
        m.header.View(),
        m.viewport.View(),
        m.footer.View(),
    )
}
```

## Copilot SDK Integration

### Session Lifecycle

```go
// 1. Create client
client, err := NewCopilotClient(WithModel("gpt-4"))
if err != nil {
    return err
}
defer client.Close()

// 2. Create session
session, err := client.CreateSession(ctx)
if err != nil {
    return err
}

// 3. Use session
events, err := client.SendPrompt(ctx, "Do something")
if err != nil {
    return err
}

// 4. Process events
for event := range events {
    // Handle event
}

// 5. Clean up
err = client.DestroySession(ctx)
```

### Event Streaming

Handle events as they arrive:

```go
events, err := client.SendPrompt(ctx, prompt)
if err != nil {
    return err
}

for event := range events {
    switch e := event.(type) {
    case *TextEvent:
        fmt.Print(e.Text)

    case *ToolCallEvent:
        result := executeTool(e.ToolCall)
        // Send result back

    case *ErrorEvent:
        return e.Error

    case *ResponseCompleteEvent:
        return nil
    }
}
```

### Tool Registration

Register custom tools:

```go
tool := Tool{
    Name:        "read_file",
    Description: "Read a file",
    Parameters: map[string]ParameterSpec{
        "path": {Type: "string", Required: true},
    },
    Handler: func(ctx context.Context, params map[string]interface{}) (string, error) {
        path := params["path"].(string)
        data, err := os.ReadFile(path)
        if err != nil {
            return "", err
        }
        return string(data), nil
    },
}

err := client.RegisterTool(tool)
```

## Testing Guidelines

### Unit Tests

Test individual functions in isolation:

```go
func TestLoopEngine_StateTransitions(t *testing.T) {
    engine := NewLoopEngine(&Config{})

    assert.Equal(t, StateIdle, engine.State())

    engine.Start()
    assert.Equal(t, StateRunning, engine.State())
}
```

### Mock Dependencies

Use interfaces and mocks:

```go
type MockSDKClient struct {
    mock.Mock
}

func (m *MockSDKClient) SendPrompt(ctx context.Context, prompt string) (<-chan Event, error) {
    args := m.Called(ctx, prompt)
    return args.Get(0).(<-chan Event), args.Error(1)
}

// In test
mockSDK := new(MockSDKClient)
mockSDK.On("SendPrompt", mock.Anything, "test").Return(events, nil)

engine := NewLoopEngine(config, mockSDK)
```

### Integration Tests

Test component interactions:

```go
func TestFullLoop(t *testing.T) {
    // Skip if no SDK access
    if os.Getenv("COPILOT_TOKEN") == "" {
        t.Skip("COPILOT_TOKEN not set")
    }

    // Real integration test
    engine := NewLoopEngine(config)
    result := engine.Run()

    assert.Equal(t, StateComplete, result.State)
}
```

### TUI Tests

Test Bubble Tea models:

```go
func TestRunnerModelUpdate(t *testing.T) {
    m := NewRunnerModel(config)

    // Test key input
    m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
    assert.NotNil(t, cmd)

    // Test custom message
    m, cmd = m.Update(LogMsg{Level: "INFO", Message: "test"})
    assert.Len(t, m.logs, 1)
}
```

## Error Handling

### Error Types

Define custom error types for different categories:

```go
type ConfigError struct {
    Field string
    Value interface{}
    Err   error
}

func (e *ConfigError) Error() string {
    return fmt.Sprintf("config error in %s: %v: %v", e.Field, e.Value, e.Err)
}

func (e *ConfigError) Unwrap() error {
    return e.Err
}
```

### Error Wrapping Chain

Build context with error wrapping:

```go
func (e *LoopEngine) executeIteration() error {
    if err := e.sendPrompt(); err != nil {
        return fmt.Errorf("iteration %d failed: %w", e.iteration, err)
    }
    return nil
}

func (e *LoopEngine) sendPrompt() error {
    if err := e.sdk.SendPrompt(); err != nil {
        return fmt.Errorf("failed to send prompt: %w", err)
    }
    return nil
}

// Results in: "iteration 3 failed: failed to send prompt: connection refused"
```

### Error Recovery

Implement retry with backoff:

```go
func (e *LoopEngine) withRetry(fn func() error) error {
    backoff := []time.Duration{1 * time.Second, 2 * time.Second, 5 * time.Second}

    var lastErr error
    for i := 0; i <= len(backoff); i++ {
        lastErr = fn()
        if lastErr == nil {
            return nil
        }

        if !isRetryable(lastErr) {
            return lastErr
        }

        if i < len(backoff) {
            time.Sleep(backoff[i])
        }
    }

    return fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

## Code Style

### Naming Conventions

- **Packages:** Lowercase, single word (e.g., `core`, `cli`, `tui`)
- **Types:** PascalCase (e.g., `LoopEngine`, `Config`)
- **Functions:** PascalCase for exported, camelCase for unexported
- **Variables:** camelCase (e.g., `maxIterations`, `promisePhrase`)
- **Constants:** PascalCase or UPPER_CASE for enum-like values
- **Interfaces:** -er suffix when appropriate (e.g., `Runner`, `Handler`)

### File Organization

```go
// 1. Package doc comment
// Package core implements...
package core

// 2. Imports (grouped)
import (
    // Standard library
    "context"
    "fmt"

    // External dependencies
    "github.com/spf13/cobra"

    // Internal packages
    "github.com/JanDeDobbeleer/copilot-ralph/internal/sdk"
)

// 3. Constants
const (
    DefaultMaxIterations = 10
    DefaultTimeout       = 30 * time.Minute
)

// 4. Types
type LoopEngine struct { }

// 5. Constructors
func NewLoopEngine() *LoopEngine { }

// 6. Methods
func (e *LoopEngine) Start() error { }

// 7. Functions
func validateConfig() error { }
```

### Comments

- **Package:** Document purpose and usage
- **Exported types:** Document behavior and usage
- **Exported functions:** Document parameters and return values
- **Complex logic:** Explain why, not what

```go
// LoopEngine orchestrates iterative AI development loops.
// It manages state transitions, detects completion promises,
// and emits events for UI updates.
//
// See specs/loop-engine.md for detailed specification.
type LoopEngine struct { }

// detectPromise checks if the given text contains the promise phrase.
// The search is case-insensitive and strips punctuation.
func (e *LoopEngine) detectPromise(text string) bool { }
```

### Line Length

- Keep lines under 100 characters when possible
- Break long function calls across multiple lines
- Use `gofmt` for automatic formatting

### Imports

- Group imports: stdlib, external, internal
- Use blank lines between groups
- Sort alphabetically within groups
- Remove unused imports

## Quick Reference

### MANDATORY: Testing Requirements

**Every code change MUST include corresponding tests:**

- New functions → Unit tests covering happy path + edge cases
- New packages → Create `*_test.go` file
- Bug fixes → Add regression test
- Refactoring → Ensure tests pass, add new coverage

**Before marking work complete:**

1. Run `go test ./...` - All tests must pass
2. Run `go build ./...` - Build must succeed
3. Check for lint/compile errors
4. Verify test coverage for new code

### Common Tasks

**Add a new command:**

1. Create handler in `internal/cli/`
2. Register with root command
3. Add flags and validation
4. Implement business logic in `internal/core/`
5. **Write tests for handler and business logic**

**Add a new TUI screen:**

1. Create model struct with state
2. Implement `Init()`, `Update()`, `View()`
3. Define custom messages
4. Add to screen flow
5. **Write Bubble Tea model tests**

**Add a new CLI flag:**

1. Add flag variable in command file
2. Register flag with command
3. Add validation if needed
4. Use flag value in logic
5. Update help text
6. **Write tests for flag handling**

**Add a custom tool:**

1. Define tool struct
2. Implement handler function
3. Register with SDK client
4. **Write tool handler tests**

### Useful Commands

```bash
# Run tests
go test ./...

# Run with race detector
go test -race ./...

# Build
go build -o ralph ./cmd/ralph

# Format code
gofmt -w .

# Run linter
golangci-lint run

# Tidy dependencies
go mod tidy
```

## Resources

- [Go Documentation](https://go.dev/doc/)
- [Bubble Tea Tutorial](https://github.com/charmbracelet/bubbletea/tree/master/tutorials)
- [Cobra User Guide](https://github.com/spf13/cobra/blob/main/user_guide.md)
- [GitHub Copilot SDK](https://github.com/github/copilot-sdk)
