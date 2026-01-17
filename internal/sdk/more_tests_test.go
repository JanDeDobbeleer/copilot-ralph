package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRetryableErrorEdgeCases(t *testing.T) {
	// Should return false for unrelated errors
	assert.False(t, isRetryableError(assert.AnError))

	// Errors containing EOF should be retryable
	assert.True(t, isRetryableError(errorString("unexpected EOF")))

	// Custom timeout string
	assert.True(t, isRetryableError(errorString("timeout occurred")))
}

// helper type to provide Error() string
type errorString string

func (e errorString) Error() string { return string(e) }
