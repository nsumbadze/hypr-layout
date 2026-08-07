package cli

import (
	"github.com/nsumbadze/hypr-layout/internal/profile"
	"github.com/nsumbadze/hypr-layout/internal/ui"
	"strings"
)

// renderPreview is used by non-interactive flows (quick, apply).
func renderPreview(layoutName string, lines []string) string {
	title := ui.Title.Render("Selected: " + layoutName)
	inner := strings.Join(lines, "\n")
	box := ui.PreviewBox.Copy().Width(60).Render(ui.Muted.Render(inner))
	return "\n  " + title + "\n\n  " + box + "\n\n"
}

// renderProfileList is used by `hypr-layout list`.
func renderProfileList(profiles []profile.Summary) string {
	if len(profiles) == 0 {
		return "\n  " + ui.Title.Render("Saved profiles") + "\n\n  " + ui.Dimmed.Render("No saved profiles.") + "\n\n"
	}
	var b strings.Builder
	b.WriteString("\n  " + ui.Title.Render("Saved profiles") + "\n\n")
	for _, p := range profiles {
		b.WriteString("  " + ui.Accent.Render("·") + "  " + ui.Base.Render(p.Name))
		if details := profileDetails(p.Meta); details != "" {
			b.WriteString("  " + ui.Dimmed.Render(details))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

// profileDetails builds a one-line metadata summary for the profile list.
func profileDetails(meta profile.Metadata) string {
	var parts []string
	if meta.Direction != "" {
		parts = append(parts, meta.Direction)
	}
	if len(meta.Monitors) > 0 {
		parts = append(parts, strings.Join(meta.Monitors, ", "))
	}
	if !meta.SavedAt.IsZero() {
		parts = append(parts, "saved "+meta.SavedAt.Local().Format("2006-01-02"))
	}
	return strings.Join(parts, "  ·  ")
}

// renderInlineStatus renders a status line for non-TUI flows.
func renderInlineStatus(icon, msg string) string {
	return "\n  " + icon + "  " + msg + "\n"
}

// renderInlineError renders a styled error box for non-TUI flows.
func renderInlineError(msg string) string {
	box := ui.ErrorBox.Copy().Padding(0, 1).Render(
		ui.Error.Copy().Bold(true).Render("Error") + "\n\n" + ui.Base.Render(msg),
	)
	return "\n  " + box + "\n"
}

// renderStatusDim renders a dim status line for non-TUI flows.
func renderStatusDim(msg string) string {
	return ui.Dimmed.Render("  " + msg)
}

// The following functions back the stdio prompt helpers in layout.go, modes.go,
// and positioning.go (still used by their unit tests and the prompt codepath).

// ── direction layout preview ──────────────────────────────────────────────────

// layoutPreviewEntry holds the visual position and display data for one monitor
