# Go-Specific Instructions for Ralph Development

This file contains Go-specific conventions and patterns for Ralph development.

## Go Version

- **Minimum:** Go 1.22
- **Features:** Use modern Go features (generics, any, etc.)

## Code Style

### No Else Statements

**Rule:** Never use `else` statements. Use early returns instead.

**Rationale:** Reduces nesting, improves readability, makes error handling clearer.

```go
// ✅ Good - Early return
func validate(data string) error {
    if data == "" {
        return errors.New("data cannot be empty")
    }
    
    if len(data) > 1000 {
        return errors.New("data too long")
    }
    
    return nil
}

// ❌ Bad - Using else
func validate(data string) error {
    if data == "" {
        return errors.New("data cannot be empty")
    } else {
        if len(data) > 1000 {
            return errors.New("data too long")
        } else {
            return nil
        }
    }
}

// ✅ Good - Early return with positive check
func isValid(value int) bool {
    if value < 0 {
        return false
    }
    
    if value > 100 {
        return false
    }
    
    return true
}

// ❌ Bad - Unnecessary else
func isValid(value int) bool {
    if value >= 0 && value <= 100 {
        return true
    } else {
        return false
    }
}
```

### Error Handling

**Always wrap errors with `%w` for error chains:**

```go
// ✅ Good - Error wrapping
if err := os.Remove(file); err != nil {
    return fmt.Errorf("failed to remove file %s: %w", file, err)
}

// ❌ Bad - No wrapping
if err := os.Remove(file); err != nil {
    return err
}

// ❌ Bad - String formatting (breaks error chains)
if err := os.Remove(file); err != nil {
    return fmt.Errorf("failed to remove file: %v", err)
}
```

**Never ignore errors:**

```go
// ✅ Good
if err := file.Close(); err != nil {
    log.Printf("Warning: failed to close file: %v", err)
}

// ❌ Bad
file.Close()  // Ignoring error

// ❌ Bad
_ = file.Close()  // Explicitly ignoring
```

**Check errors immediately:**

```go
// ✅ Good
data, err := readFile(path)
if err != nil {
    return fmt.Errorf("read failed: %w", err)
}
processData(data)

// ❌ Bad - Deferred error checking
data, err := readFile(path)
result := processData(data)
if err != nil {
    return err
}
```

### Package Documentation

Every package must have a documentation comment:

```go
// Package core implements the Ralph loop engine.
//
// The loop engine orchestrates iterative AI development loops,
// managing state transitions, promise detection, and event emission.
//
// See specs/loop-engine.md for detailed specification.
package core
```

### Exported Symbol Documentation

Document all exported types, functions, and constants:

```go
// LoopEngine orchestrates iterative AI development loops.
// It manages state transitions, detects completion promises,
// and emits events for UI updates.
type LoopEngine struct {
    // unexported fields
}

// NewLoopEngine creates a new loop engine with the given configuration.
// It returns an error if the configuration is invalid.
func NewLoopEngine(config *Config) (*LoopEngine, error) {
    // ...
}

// Start begins loop execution.
// It returns an error if the loop is already running or if initialization fails.
func (e *LoopEngine) Start() error {
    // ...
}
```

### Naming Conventions

```go
// ✅ Good naming
type LoopEngine struct { }        // Type: PascalCase
const DefaultTimeout = 30         // Constant: PascalCase
var maxIterations int             // Variable: camelCase
func processData() { }            // Unexported: camelCase
func ValidateConfig() { }         // Exported: PascalCase

// Interface with -er suffix
type Runner interface {
    Run() error
}

// ❌ Bad naming
type loop_engine struct { }       // Don't use snake_case
const default_timeout = 30        // Don't use snake_case
var MaxIterations int             // Don't export unless necessary
```

## Testing

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
            name:     "partial match",
            text:     "The task is done!",
            promise:  "done!",
            expected: true,
        },
        {
            name:     "not found",
            text:     "still working",
            promise:  "done!",
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

### Test Structure

```go
func TestFunctionName(t *testing.T) {
    // Arrange - Set up test data
    config := &Config{MaxIterations: 10}
    engine := NewLoopEngine(config)
    
    // Act - Execute the function
    result, err := engine.Run()
    
    // Assert - Check results
    require.NoError(t, err)
    assert.Equal(t, StateComplete, result.State)
}
```

### Subtests

Use subtests for related scenarios:

```go
func TestLoopEngine(t *testing.T) {
    t.Run("state transitions", func(t *testing.T) {
        // Test state transitions
    })
    
    t.Run("error handling", func(t *testing.T) {
        // Test error cases
    })
    
    t.Run("promise detection", func(t *testing.T) {
        // Test promise detection
    })
}
```

### Mock Interfaces

Use interfaces and mocks for testing:

```go
// Define interface
type SDKClient interface {
    SendPrompt(ctx context.Context, prompt string) error
}

// Mock implementation
type MockSDKClient struct {
    mock.Mock
}

func (m *MockSDKClient) SendPrompt(ctx context.Context, prompt string) error {
    args := m.Called(ctx, prompt)
    return args.Error(0)
}

// Use in test
func TestWithMock(t *testing.T) {
    mockSDK := new(MockSDKClient)
    mockSDK.On("SendPrompt", mock.Anything, "test").Return(nil)
    
    engine := NewLoopEngine(config, mockSDK)
    err := engine.Run()
    
    assert.NoError(t, err)
    mockSDK.AssertExpectations(t)
}
```

## Bubble Tea Specific

### Model-View-Update Pattern

```go
type Model struct {
    state   State
    items   []string
    cursor  int
}

// Init is called once when the program starts
func (m Model) Init() tea.Cmd {
    return nil  // Return initial command or nil
}

// Update handles messages and updates state
// Never use else in Update!
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        case "up":
            if m.cursor > 0 {
                m.cursor--
            }
            return m, nil
        case "down":
            if m.cursor < len(m.items)-1 {
                m.cursor++
            }
            return m, nil
        }
    
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        return m, nil
    }
    
    return m, nil
}

// View renders the model to a string
func (m Model) View() string {
    var s strings.Builder
    
    for i, item := range m.items {
        if i == m.cursor {
            s.WriteString("> ")
        } else {
            s.WriteString("  ")
        }
        s.WriteString(item)
        s.WriteString("\n")
    }
    
    return s.String()
}
```

### Commands

Commands return messages asynchronously:

```go
// Simple command
func tickCmd() tea.Msg {
    time.Sleep(time.Second)
    return tickMsg{}
}

// Command with work
func fetchDataCmd(url string) tea.Cmd {
    return func() tea.Msg {
        data, err := fetch(url)
        if err != nil {
            return errMsg{err}
        }
        return dataMsg{data}
    }
}

// Use in Update
case startMsg:
    return m, fetchDataCmd("https://api.example.com")
```

### Message Types

Define custom message types:

```go
type tickMsg struct{}

type dataMsg struct {
    data []byte
}

type errMsg struct {
    err error
}

// In Update
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case dataMsg:
        m.data = msg.data
        return m, nil
        
    case errMsg:
        m.err = msg.err
        return m, nil
    }
    
    return m, nil
}
```

## Concurrency

### Channel Patterns

```go
// Producer-consumer
func producer(out chan<- Event) {
    defer close(out)
    for {
        out <- Event{}
    }
}

func consumer(in <-chan Event) {
    for event := range in {
        process(event)
    }
}

// Select with context
func process(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case event := <-events:
            handle(event)
        }
    }
}
```

### Context Usage

```go
// Always accept context as first parameter
func doWork(ctx context.Context, data string) error {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Continue work
    return nil
}

// Pass context to downstream calls
func orchestrate(ctx context.Context) error {
    if err := doWork(ctx, "data"); err != nil {
        return fmt.Errorf("work failed: %w", err)
    }
    return nil
}
```

## Performance

### String Building

```go
// ✅ Good - Use strings.Builder
var b strings.Builder
for _, item := range items {
    b.WriteString(item)
    b.WriteString("\n")
}
result := b.String()

// ❌ Bad - Inefficient concatenation
result := ""
for _, item := range items {
    result += item + "\n"
}
```

### Avoid Premature Optimization

- Write clear code first
- Profile before optimizing
- Focus on algorithmic improvements
- Use built-in optimizations (e.g., strings.Builder, sync.Pool)

## Common Pitfalls

### Loop Variable Capture

```go
// ❌ Bad - Variable captured in goroutine
for _, item := range items {
    go func() {
        process(item)  // Wrong! All goroutines see last item
    }()
}

// ✅ Good - Pass variable
for _, item := range items {
    go func(i Item) {
        process(i)
    }(item)
}

// ✅ Good - Shadow variable (Go 1.22+)
for _, item := range items {
    item := item  // Create new variable
    go func() {
        process(item)
    }()
}
```

### Pointer Receivers

```go
// Use pointer receivers when:
// 1. Method modifies receiver
// 2. Receiver is large struct
// 3. Other methods use pointer receiver (consistency)

// ✅ Modifies state
func (e *LoopEngine) Start() error {
    e.state = StateRunning
    return nil
}

// ✅ Read-only, but consistent with pointer methods
func (e *LoopEngine) State() LoopState {
    return e.state
}
```

## Resources

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
