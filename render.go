package main

import (
	"fmt"
	"strconv"
	"strings"
)

// renderPreview is used by non-interactive flows (quick, apply).
func renderPreview(layoutName string, lines []string) string {
	title := styleTitle.Render("Selected: " + layoutName)
	inner := strings.Join(lines, "\n")
	box := stylePreviewBox.Copy().Width(60).Render(styleMuted.Render(inner))
	return "\n  " + title + "\n\n  " + box + "\n\n"
}

// renderProfileList is used by `hypr-layout list`.
func renderProfileList(profiles []string) string {
	if len(profiles) == 0 {
		return "\n  " + styleTitle.Render("Saved profiles") + "\n\n  " + styleDimmed.Render("No saved profiles.") + "\n\n"
	}
	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("Saved profiles") + "\n\n")
	for _, p := range profiles {
		b.WriteString("  " + styleAccent.Render("·") + "  " + styleBase.Render(p) + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

// renderInlineStatus renders a status line for non-TUI flows.
func renderInlineStatus(icon, msg string) string {
	return "\n  " + icon + "  " + msg + "\n"
}

// renderInlineError renders a styled error box for non-TUI flows.
func renderInlineError(msg string) string {
	box := styleErrorBox.Copy().Padding(0, 1).Render(
		styleErr.Copy().Bold(true).Render("Error") + "\n\n" + styleBase.Render(msg),
	)
	return "\n  " + box + "\n"
}

// renderStatusDim renders a dim status line for non-TUI flows.
func renderStatusDim(msg string) string {
	return styleDimmed.Render("  " + msg)
}

// The following functions back the stdio prompt helpers in layout.go, modes.go,
// and positioning.go (still used by their unit tests and the prompt codepath).

func renderLayoutOptions(options []layoutOption) string {
	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("Layout options") + "\n\n")
	for _, option := range options {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			styleDimmed.Render(fmt.Sprintf("%d.", option.ID)),
			styleBase.Render(option.Name),
		))
	}
	b.WriteString("\n  " + stylePromptGlyph.Render("❯ ") + styleMuted.Render("Select a layout: "))
	return b.String()
}

func renderMonitorModes(mon monitor, current monitorMode, modes []monitorMode) string {
	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("Modes for "+mon.Name) + "\n")
	b.WriteString("  " + styleMuted.Render("Current: "+formatMonitorMode(current)) + "\n\n")
	for i, mode := range modes {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			styleDimmed.Render(fmt.Sprintf("%d.", i+1)),
			styleBase.Render(formatMonitorMode(mode)),
		))
	}
	b.WriteString("\n  " + stylePromptGlyph.Render("❯ ") + styleMuted.Render("Select a mode (Enter keeps current): "))
	return b.String()
}

func renderLayoutDirections() string {
	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("Layout direction") + "\n\n")
	b.WriteString("  " + styleDimmed.Render("1.") + "  " + styleBase.Render("Horizontal  (left → right)") + "\n")
	b.WriteString("  " + styleDimmed.Render("2.") + "  " + styleBase.Render("Vertical    (top → bottom)") + "\n")
	b.WriteString("\n  " + stylePromptGlyph.Render("❯ ") + styleMuted.Render("Select layout direction: "))
	return b.String()
}

func renderMonitorOrderPrompt(monitors []monitor, activeConfigs []activeMonitorConfig) string {
	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("Active monitors") + "\n\n")
	for i, config := range activeConfigs {
		mon := monitors[config.Index]
		b.WriteString(fmt.Sprintf("  %s  %s  %s\n",
			styleDimmed.Render(fmt.Sprintf("(%d)", i+1)),
			styleBase.Render(mon.Name),
			styleDimmed.Render(formatMonitorMode(config.Mode)),
		))
	}
	b.WriteString("\n  " + stylePromptGlyph.Render("❯ ") + styleMuted.Render("Enter monitor order by indices (e.g. \"2 1 3\"): "))
	return b.String()
}

// ── shared formatting ─────────────────────────────────────────────────────────

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
