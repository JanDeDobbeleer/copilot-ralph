package sdk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventTypesAndConstructors(t *testing.T) {
	te := NewTextEvent("hello")
	assert.Equal(t, EventTypeText, te.Type())
	assert.Equal(t, "hello", te.Text)
	assert.WithinDuration(t, time.Now(), te.Timestamp(), time.Second)

	tc := ToolCall{ID: "1", Name: "test"}
	tce := NewToolCallEvent(tc)
	assert.Equal(t, EventTypeToolCall, tce.Type())
	assert.Equal(t, "test", tce.ToolCall.Name)

	r := NewToolResultEvent(tc, "res", nil)
	assert.Equal(t, EventTypeToolResult, r.Type())
	assert.Equal(t, "res", r.Result)

	e := NewErrorEvent(nil)
	assert.Equal(t, EventTypeError, e.Type())
	assert.Equal(t, "", e.Error())

	err := NewErrorEvent(assert.AnError)
	assert.NotEmpty(t, err.Error())
	assert.WithinDuration(t, time.Now(), err.Timestamp(), time.Second)
}
