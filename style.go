package main

import "github.com/charmbracelet/lipgloss"

// Adaptive color palette — one accent, muted, error, success, dim.
// Nothing outside this file should hardcode colors.
var (
	colorAccent  = lipgloss.AdaptiveColor{Dark: "#7D56F4", Light: "#6D28D9"}
	colorMuted   = lipgloss.AdaptiveColor{Dark: "#9CA3AF", Light: "#6B7280"}
	colorDim     = lipgloss.AdaptiveColor{Dark: "#4B5563", Light: "#D1D5DB"}
	colorError   = lipgloss.AdaptiveColor{Dark: "#F87171", Light: "#DC2626"}
	colorSuccess = lipgloss.AdaptiveColor{Dark: "#34D399", Light: "#059669"}
	colorText    = lipgloss.AdaptiveColor{Dark: "#E5E7EB", Light: "#1F2937"}
	colorBorder  = lipgloss.AdaptiveColor{Dark: "#374151", Light: "#D1D5DB"}
	colorBarBg   = lipgloss.AdaptiveColor{Dark: "#0F1117", Light: "#F3F4F6"}
)

var (
	styleBase    = lipgloss.NewStyle().Foreground(colorText)
	styleAccent  = lipgloss.NewStyle().Foreground(colorAccent)
	styleMuted   = lipgloss.NewStyle().Foreground(colorMuted)
	styleDimmed  = lipgloss.NewStyle().Foreground(colorDim)
	styleSuccess = lipgloss.NewStyle().Foreground(colorSuccess)
	styleErr     = lipgloss.NewStyle().Foreground(colorError)

	styleTitle = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true)

	styleSelected = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	stylePromptGlyph = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	stylePreviewBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	styleErrorBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorError).
			Padding(0, 1)

	styleBarBg = lipgloss.NewStyle().Background(colorBarBg)
)
