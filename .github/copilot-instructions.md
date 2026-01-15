# GitHub Copilot Instructions for Ralph

This file provides quick reference instructions for GitHub Copilot when working on Ralph. For comprehensive details, see [AGENTS.md](../AGENTS.md).

## Project Overview

Ralph is an iterative AI development loop tool built in Go, using:
- **CLI:** Cobra for commands
- **TUI:** Bubble Tea for terminal UI
- **AI:** GitHub Copilot SDK
- **Testing:** testify for assertions

## Key Principles

### 1. Spec-Driven Development
- **Read specs first:** All specs are in `/specs/` directory
- **Implement per spec:** Follow specifications exactly
- **Update spec first:** If requirements change, update spec before code

### 2. Code Style

See [Go-specific instructions](instructions/golang.md) for detailed Go conventions including:
- **No `else` statements** - Use early returns
- **Error wrapping** - Always use `%w`
- **Table-driven tests** - Multiple test cases in one test
- **Bubble Tea patterns** - Model-View-Update (MVU)
- **Package documentation** - Document all exported symbols

### 3. Architecture

```
CLI Layer (cmd/ralph, internal/cli)
    ↓
TUI Layer (internal/tui)
    ↓
Core Layer (internal/core)
    ↓
SDK Layer (internal/sdk)
```

**Dependency Rules:**
- CLI depends on TUI and Core
- TUI depends on Core
- Core depends on SDK
- SDK has no internal dependencies
- No circular dependencies

### 4. Bubble Tea Pattern

All TUI components use Model-View-Update. See [golang.md](instructions/golang.md) for detailed patterns:

- **Model** - State and data
- **Init** - Initialize and return initial commands
- **Update** - Handle messages and update state (no `else` statements!)
- **View** - Render to string

### 5. Testing

- **Unit tests:** Test individual functions
- **Mock dependencies:** Use interfaces
- **Table-driven:** Multiple test cases in one test
- **Integration tests:** Guard with environment checks

## File Structure

```
copilot-ralph/
├── cmd/ralph/              # Entry point
├── internal/
│   ├── cli/                # Commands
│   ├── tui/                # Terminal UI
│   │   ├── wizard/         # Initialization wizard
│   │   ├── runner/         # Loop display
│   │   ├── styles/         # Styling
│   │   └── keys/           # Key bindings
│   ├── core/               # Business logic
│   └── sdk/                # Copilot SDK wrapper
├── pkg/version/            # Version info
└── specs/                  # Specifications (READ THESE!)
```

## Common Patterns

See [golang.md](instructions/golang.md) for comprehensive patterns. Key examples:

### CLI Flags
```go
var maxIterations int
cmd.Flags().IntVarP(&maxIterations, "max-iterations", "m", 10, "maximum iterations")
```

### Event Streaming
```go
events := make(chan Event, 100)
go func() {
    defer close(events)
    // Emit events
}()
return events
```

For detailed patterns including error handling, concurrency, and Bubble Tea specifics, see the [Go-specific instructions](instructions/golang.md).

## Quick Commands

```bash
# Build
go build -o ralph ./cmd/ralph

# Test
go test -race ./...

# Format
gofmt -w .

# Tidy
go mod tidy
```

## Important References

- **Full Guide:** [AGENTS.md](../AGENTS.md)
- **Specifications:** [specs/](../specs/)
- **Go Guidelines:** [golang.md](instructions/golang.md)

## When to Check

- **Adding features:** Read relevant spec in `/specs/`
- **Fixing bugs:** Check AGENTS.md for patterns
- **Changing TUI:** Review `specs/tui.md` and Bubble Tea patterns
- **SDK integration:** See `specs/sdk-integration.md`
- **CLI flags:** See `internal/cli/run.go` for examples

## Development Workflow

1. **Understand:** Read relevant specification
2. **Design:** Define interfaces if needed
3. **Implement:** Follow spec and conventions
4. **Test:** Write table-driven tests
5. **Verify:** Run `go test ./...` and `go build ./...`
6. **Document:** Add package/function docs

## MANDATORY: Testing Requirements

**Every code change MUST include corresponding tests:**

- **New functions:** Write unit tests covering happy path + edge cases
- **New packages:** Create `*_test.go` file with package tests
- **Bug fixes:** Add regression test that would have caught the bug
- **Refactoring:** Ensure existing tests still pass, add tests for new code paths

**Before marking work complete:**

1. Run `go test ./...` - All tests must pass
2. Run `go build ./...` - Build must succeed
3. Check `get_errors` - No lint/compile errors
4. Verify test coverage for new code

**Test file naming:** `<package>_test.go` (e.g., `loop_test.go`, `client_test.go`)

## Remember

- ❌ Never use `else` statements
- ✅ Always wrap errors with `%w`
- ✅ Always check errors
- ✅ Use table-driven tests
- ✅ Follow MVU pattern for TUI
- ✅ Read specs before implementing
- ✅ Document exported symbols
- ✅ Keep packages focused
