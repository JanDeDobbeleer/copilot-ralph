package sdk

import (
	"context"
	"sync"
	"testing"
	"time"

	copilot "github.com/github/copilot-sdk/go"
	generated "github.com/github/copilot-sdk/go/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSession implements the minimal subset of copilot.Session used by client.go
type fakeSession struct {
	mu       sync.Mutex
	handlers []func(generated.SessionEvent)
	// control what Send does
	sendFunc func()
}

func (f *fakeSession) On(h func(generated.SessionEvent)) func() {
	f.mu.Lock()
	f.handlers = append(f.handlers, h)
	idx := len(f.handlers) - 1
	f.mu.Unlock()
	return func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		// remove handler
		f.handlers[idx] = nil
	}
}

func (f *fakeSession) Send(opts interface{}) (string, error) {
	// Simulate asynchronous events being emitted
	go func() {
		f.mu.Lock()
		handlers := append([]func(generated.SessionEvent){}, f.handlers...)
		f.mu.Unlock()

		// 1) streaming delta
		for _, h := range handlers {
			if h == nil {
				continue
			}
			h(generated.SessionEvent{Type: "assistant.message_delta", Data: generated.Data{DeltaContent: ptrString("Hello ")}})
		}

		// 2) final message
		for _, h := range handlers {
			if h == nil {
				continue
			}
			h(generated.SessionEvent{Type: "assistant.message", Data: generated.Data{Content: ptrString("Hello world")}})
		}

		// 3) session idle
		for _, h := range handlers {
			if h == nil {
				continue
			}
			h(generated.SessionEvent{Type: "session.idle"})
		}
	}()

	if f.sendFunc != nil {
		f.sendFunc()
	}

	return "msg-id", nil
}

func (f *fakeSession) Abort() error   { return nil }
func (f *fakeSession) Destroy() error { return nil }

func ptrString(s string) *string { return &s }

// testEventDrainTimeout is used by tests that need to drain the events channel
// without relying on the producer to close it. We use a short timeout to avoid
// indefinite blocking in tests where the code under test may return early
// (for example, when a context is canceled).
const testEventDrainTimeout = 100 * time.Millisecond

func TestSafeEventSenderOnClosedChannel(t *testing.T) {
	ch := make(chan Event)
	close(ch)
	err := safeEventSender(ch, NewTextEvent("x"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "event channel closed")
}

func TestHandleSDKEventVariousTypes(t *testing.T) {
	c := &CopilotClient{}
	events := make(chan Event, 10)
	defer close(events)
	var closed bool
	closeDone := func() { closed = true }
	pending := make(map[string]ToolCall)

	// assistant.message_delta
	c.handleSDKEvent(generated.SessionEvent{Type: "assistant.message_delta", Data: generated.Data{DeltaContent: ptrString("part")}}, events, new(string), closeDone, pending)

	// assistant.message
	c.handleSDKEvent(generated.SessionEvent{Type: "assistant.message", Data: generated.Data{Content: ptrString("full")}}, events, new(string), closeDone, pending)

	// tool.execution_start
	c.handleSDKEvent(generated.SessionEvent{Type: "tool.execution_start", Data: generated.Data{ToolName: ptrString("edit"), ToolCallID: ptrString("1"), Arguments: map[string]interface{}{"path": "a.go"}}}, events, new(string), closeDone, pending)

	// tool.execution_complete success
	// adapt to copilot.ToolResult fields
	c.handleSDKEvent(generated.SessionEvent{Type: "tool.execution_complete", Data: generated.Data{ToolCallID: ptrString("1"), ToolName: ptrString("edit"), Result: &generated.Result{Content: "ok"}, Success: ptrBool(true)}}, events, new(string), closeDone, pending)

	// tool.execution_complete failure with Error.String
	errStr := "tool failed"
	c.handleSDKEvent(generated.SessionEvent{Type: "tool.execution_complete", Data: generated.Data{ToolCallID: ptrString("2"), ToolName: ptrString("run"), Result: &generated.Result{Content: ""}, Success: ptrBool(false), Error: &generated.ErrorUnion{String: &errStr}}}, events, new(string), closeDone, pending)

	// session.error
	msg := "bad"
	c.handleSDKEvent(generated.SessionEvent{Type: "session.error", Data: generated.Data{Message: &msg}}, events, new(string), closeDone, pending)

	// session.idle should call closeDone
	c.handleSDKEvent(generated.SessionEvent{Type: "session.idle"}, events, new(string), closeDone, pending)

	// Drain events and assert some expected types
	received := []Event{}
	down := time.After(200 * time.Millisecond)
loop:
	for {
		select {
		case ev := <-events:
			received = append(received, ev)
			if len(received) >= 6 {
				break loop
			}
		case <-down:
			break loop
		}
	}

	// Expect at least one TextEvent and at least one ToolResultEvent and one ErrorEvent
	var hasText, hasToolResult, hasError bool
	for _, e := range received {
		switch e.Type() {
		case EventTypeText:
			hasText = true
		case EventTypeToolResult:
			hasToolResult = true
		case EventTypeError:
			hasError = true
		}
	}

	assert.True(t, hasText, "should have text events")
	assert.True(t, hasToolResult, "should have tool result events")
	assert.True(t, hasError, "should have error events")
	assert.True(t, closed, "closeDone should be called on session.idle")
}

func ptrBool(b bool) *bool { return &b }

func TestSendPromptOnceWithFakeSession(t *testing.T) {
	c, err := NewCopilotClient()
	require.NoError(t, err)

	// ensure client has a session wrapper to record history
	c.session = NewSession()

	events := make(chan Event, 10)
	defer close(events)

	// call sendPromptWithRetry with a canceled context to cover the cancellation early-return path
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.sendPromptWithRetry(ctx, nil, "hello", events)
	// no error expected; function returns after cancellation
	// drain any events with a short timeout to avoid indefinite blocking
	done := time.After(100 * time.Millisecond)
drainLoop:
	for {
		select {
		case _, ok := <-events:
			if !ok {
				break drainLoop
			}
			// continue draining
		case <-done:
			// timeout reached; stop draining
			break drainLoop
		}
	}

	// We canceled the context before send, so no history changes expected; just ensure function returned cleanly.
	require.Len(t, c.session.History, 0)
}

// testSessionAdapter adapts fakeSession to the concrete type expected by client.sendPromptOnce signature
// by implementing the methods used by sendPromptOnce via compatible signatures.

type testSessionAdapter struct{ inner *fakeSession }

func (a *testSessionAdapter) On(h func(copilot.SessionEvent)) func() {
	return a.inner.On(func(e generated.SessionEvent) { h(copilot.SessionEvent{Type: e.Type, Data: generated.Data{}}) })
}
func (a *testSessionAdapter) Send(opts copilot.MessageOptions) (string, error) {
	// Delegate to inner and ignore options
	return a.inner.Send(nil)
}
func (a *testSessionAdapter) Abort() error   { return a.inner.Abort() }
func (a *testSessionAdapter) Destroy() error { return a.inner.Destroy() }
