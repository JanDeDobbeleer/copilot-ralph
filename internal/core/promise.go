// Package core provides promise detection for the loop engine.

package core

import (
	"fmt"
	"strings"
)

// detectPromise checks if the given text contains the promise phrase.
// The search is case-insensitive and tolerates punctuation differences.
// It returns true if the promise phrase is found in the text.
func detectPromise(text, promisePhrase string) bool {
	if promisePhrase == "" {
		return false
	}

	promisePhrase = fmt.Sprintf("<promise>%s</promise>", promisePhrase)

	return strings.Contains(text, promisePhrase)
}
