// Package ui holds the colour palette and text helpers shared by the command
// line output and the wizard, so nothing else hardcodes a colour.
package ui

import "github.com/charmbracelet/lipgloss"

// Adaptive color palette — one accent, muted, error, success, dim.
// Nothing outside this file should hardcode colors.
var (
	ColorAccent  = lipgloss.AdaptiveColor{Dark: "#7D56F4", Light: "#6D28D9"}
	colorMuted   = lipgloss.AdaptiveColor{Dark: "#9CA3AF", Light: "#6B7280"}
	colorDim     = lipgloss.AdaptiveColor{Dark: "#4B5563", Light: "#D1D5DB"}
	colorError   = lipgloss.AdaptiveColor{Dark: "#F87171", Light: "#DC2626"}
	colorSuccess = lipgloss.AdaptiveColor{Dark: "#34D399", Light: "#059669"}
	colorText    = lipgloss.AdaptiveColor{Dark: "#E5E7EB", Light: "#1F2937"}
	ColorBorder  = lipgloss.AdaptiveColor{Dark: "#374151", Light: "#D1D5DB"}
	colorBarBg   = lipgloss.AdaptiveColor{Dark: "#0F1117", Light: "#F3F4F6"}
)

var (
	Base    = lipgloss.NewStyle().Foreground(colorText)
	Accent  = lipgloss.NewStyle().Foreground(ColorAccent)
	Muted   = lipgloss.NewStyle().Foreground(colorMuted)
	Dimmed  = lipgloss.NewStyle().Foreground(colorDim)
	Success = lipgloss.NewStyle().Foreground(colorSuccess)
	Error   = lipgloss.NewStyle().Foreground(colorError)

	Title = lipgloss.NewStyle().
		Foreground(colorText).
		Bold(true)

	Selected = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	PromptGlyph = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	PreviewBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	ErrorBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorError).
			Padding(0, 1)

	BarBg = lipgloss.NewStyle().Background(colorBarBg)
)
