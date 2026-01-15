// Package styles provides Lip Gloss styling for TUI components.
//
// This package defines all colors, borders, and text styles used
// throughout the Ralph TUI for consistent visual design.
package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette - Lip Gloss vibrant theme
var (
	// Primary colors
	Primary = lipgloss.Color("#ff1cf0") // Hot Pink (main brand color)
	Success = lipgloss.Color("#50fa7b") // Bright Green (checkmarks, success)
	Warning = lipgloss.Color("#f9e2af") // Yellow (warnings)
	Error   = lipgloss.Color("#f38ba8") // Red (errors)
	Info    = lipgloss.Color("#00d9ff") // Bright Cyan
)

// Title styles
var (
	// TitleStyle is for main screen titles.
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary).
		MarginBottom(1)
)

// Message styles
var (
	// PrimaryStyle for primary text.
	PrimaryStyle = lipgloss.NewStyle().Foreground(Primary)
	// InfoStyle for informational messages.
	InfoStyle = lipgloss.NewStyle().Foreground(Info)
	// SuccessStyle for success messages.
	SuccessStyle = lipgloss.NewStyle().Foreground(Success)
	// WarningStyle for warning messages.
	WarningStyle = lipgloss.NewStyle().Foreground(Warning)
	// ErrorStyle for error messages.
	ErrorStyle = lipgloss.NewStyle().Foreground(Error)
)
