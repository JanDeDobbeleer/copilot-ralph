// Package core provides promise detection for the loop engine.

package core

import (
	"regexp"
	"strings"
)

// punctuationPattern matches punctuation characters for stripping.
var punctuationPattern = regexp.MustCompile(`[^\w\s]`)

// detectPromise checks if the given text contains the promise phrase.
// The search is case-insensitive and tolerates punctuation differences.
// It returns true if the promise phrase is found in the text.
func detectPromise(text, promisePhrase string) bool {
	if promisePhrase == "" {
		return false
	}

	// Normalize both strings: lowercase and trim whitespace
	normalizedText := strings.ToLower(strings.TrimSpace(text))
	normalizedPhrase := strings.ToLower(strings.TrimSpace(promisePhrase))

	// Check for exact match (case-insensitive)
	if strings.Contains(normalizedText, normalizedPhrase) {
		return true
	}

	// Check for match without punctuation
	textNoPunct := punctuationPattern.ReplaceAllString(normalizedText, "")
	phraseNoPunct := punctuationPattern.ReplaceAllString(normalizedPhrase, "")

	if phraseNoPunct == "" {
		return false
	}

	// Normalize whitespace in punctuation-stripped versions
	textNoPunct = normalizeWhitespace(textNoPunct)
	phraseNoPunct = normalizeWhitespace(phraseNoPunct)

	return strings.Contains(textNoPunct, phraseNoPunct)
}

// normalizeWhitespace collapses multiple whitespace characters into single spaces.
func normalizeWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
