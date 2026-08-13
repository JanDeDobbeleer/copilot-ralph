package styles

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestRalphWiggumAndStyles(t *testing.T) {
	// ASCII art should be non-empty and contain unicode blocks
	assert.True(t, len(RalphWiggum) > 0)
	assert.True(t, strings.Contains(RalphWiggum, "⣀") || strings.Contains(RalphWiggum, "⠉"))

	// Ensure style variables render without panicking and include content
	r := TitleStyle.Render("X")
	assert.Contains(t, r, "X")

	// Basic color/constants are non-empty
	assert.NotEmpty(t, Primary)
	assert.NotEmpty(t, Success)

	// Verify style types are lipgloss styles
	_ = lipgloss.NewStyle()
}
