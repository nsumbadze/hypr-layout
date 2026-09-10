package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
	"github.com/nsumbadze/hypr-layout/internal/layout"
	"github.com/nsumbadze/hypr-layout/internal/profile"
	"github.com/nsumbadze/hypr-layout/internal/ui"
)

// ── view ──────────────────────────────────────────────────────────────────────

func (m tuiModel) View() string {
	if m.width == 0 {
		return ""
	}
	if m.width < 40 || m.height < 10 {
		return "  Terminal too small — resize to continue.\n"
	}
	// MaxHeight clips any component quirk (e.g. the list paginator drawing one
	// line beyond its set height) so the footer is never pushed off-screen.
	contentH := m.height - 2
	body := lipgloss.NewStyle().Height(contentH).MaxHeight(contentH).Render(m.bodyView())
	return lipgloss.JoinVertical(lipgloss.Left,
		m.headerView(),
		body,
		m.footerView(),
	)
}

func (m tuiModel) headerView() string {
	appName := ui.Accent.Copy().Bold(true).Render("hypr-layout")
	step := ui.Muted.Render(m.stepLabel())
	inner := "  " + appName + "  " + ui.Dimmed.Render("·") + "  " + step
	return m.barLine(inner)
}

func (m tuiModel) footerView() string {
	return m.barLine("  " + m.footerHints())
}

// barLine pads a header/footer line to the full width and truncates it to a
// single line so narrow terminals never wrap it onto a second row.
func (m tuiModel) barLine(inner string) string {
	pad := m.width - lipgloss.Width(inner)
	if pad > 0 {
		inner += strings.Repeat(" ", pad)
	}
	return ui.BarBg.Copy().MaxWidth(m.width).MaxHeight(1).Render(inner)
}

func (m tuiModel) stepLabel() string {
	switch m.state {
	case tuiDetecting:
		return "Detecting monitors"
	case tuiLayoutSelect:
		return "Select layout"
	case tuiReview:
		return "Review layout"
	case tuiModeSelect:
		if mon, ok := m.editedMonitor(); ok {
			return "Mode for " + mon.Name
		}
		return "Select mode"
	case tuiTransformSelect:
		if mon, ok := m.editedMonitor(); ok {
			return "Rotation for " + mon.Name
		}
		return "Rotation"
	case tuiVRRSelect:
		if mon, ok := m.editedMonitor(); ok {
			return "VRR for " + mon.Name
		}
		return "VRR"
	case tuiDirectionSelect:
		return "Layout direction"
	case tuiOrderEdit:
		return "Monitor order"
	case tuiConfigView:
		return "Generated config"
	case tuiProfileName:
		return "Profile name"
	case tuiApplying:
		return "Writing config…"
	case tuiReloading:
		return "Reloading…"
	case tuiDone:
		return "Done"
	case tuiErr:
		return "Error"
	default:
		return ""
	}
}

func (m tuiModel) footerHints() string {
	dim := func(s string) string { return ui.Dimmed.Render(s) }
	acc := func(s string) string { return ui.Accent.Render(s) }
	sep := ui.Dimmed.Render(" · ")
	back := acc("esc") + " back"
	switch m.state {
	case tuiLayoutSelect:
		return dim("↑/↓") + " navigate" + sep + acc("enter") + " select" + sep + acc("q") + " quit"
	case tuiReview:
		return dim("↑/↓") + " move" + sep + acc("enter") + " change" + sep +
			acc("a") + " apply" + sep + back + sep + acc("q") + " quit"
	case tuiModeSelect, tuiTransformSelect, tuiVRRSelect, tuiDirectionSelect:
		return dim("↑/↓") + " navigate" + sep + acc("enter") + " select" + sep + back + sep + acc("q") + " quit"
	case tuiOrderEdit:
		return dim("↑/↓") + " select" + sep + acc("shift+↑/↓") + " move" + sep +
			acc("enter") + " done" + sep + acc("esc") + " cancel"
	case tuiConfigView:
		return dim("↑/↓") + " scroll" + sep + acc("enter") + " done" + sep + acc("q") + " quit"
	case tuiProfileName:
		return acc("enter") + " save" + sep + back + sep + acc("q") + " quit"
	case tuiDone, tuiErr:
		return dim("any key") + " exit"
	default:
		return ""
	}
}

func (m tuiModel) bodyView() string {
	contentH := m.height - 2 // one line each for header and footer
	switch m.state {
	case tuiDetecting:
		return m.spinnerView(contentH, "Detecting monitors…")
	case tuiLayoutSelect:
		return m.layoutSelectView(contentH)
	case tuiReview:
		return m.reviewView(contentH)
	case tuiModeSelect:
		return m.modeSelectView(contentH)
	case tuiTransformSelect:
		return m.transformSelectView(contentH)
	case tuiVRRSelect:
		return m.vrrSelectView(contentH)
	case tuiDirectionSelect:
		return m.directionSelectView(contentH)
	case tuiOrderEdit:
		return m.orderEditView(contentH)
	case tuiConfigView:
		return m.configView(contentH)
	case tuiProfileName:
		return m.profileNameView(contentH)
	case tuiApplying:
		return m.spinnerView(contentH, "Writing config…")
	case tuiReloading:
		return m.spinnerView(contentH, "Reloading Hyprland…")
	case tuiDone:
		return m.doneView(contentH)
	case tuiErr:
		return m.errView(contentH)
	default:
		return ""
	}
}

// profileMetadata captures how the current layout was produced for the
// profile header: direction (or "mirror"), and the ordered monitors with
// their chosen modes.
func (m tuiModel) profileMetadata() profile.Metadata {
	meta := profile.Metadata{
		SavedAt:   time.Now(),
		Direction: string(m.direction),
	}
	if m.mirrored {
		meta.Direction = "mirror"
	}
	for _, cfg := range m.activeConfigs {
		meta.Monitors = append(meta.Monitors, m.monitors[cfg.Index].Name+" "+layout.FormatMode(cfg.Mode))
	}
	return meta
}

// editedMonitor returns the monitor an editor screen is acting on.
func (m tuiModel) editedMonitor() (hypr.Monitor, bool) {
	if m.editIdx < 0 || m.editIdx >= len(m.activeConfigs) {
		return hypr.Monitor{}, false
	}
	return m.monitors[m.activeConfigs[m.editIdx].Index], true
}

// ── body sub-views ────────────────────────────────────────────────────────────

func (m tuiModel) spinnerView(h int, label string) string {
	// Small logo + spinner, shown during detect/apply/reload
	var b strings.Builder
	b.WriteString("\n\n")
	if m.state == tuiDetecting {
		b.WriteString("  " + ui.Dimmed.Copy().Bold(true).Render("◆ hypr-layout") + "\n")
		b.WriteString("  " + ui.Dimmed.Render("monitor layout manager") + "\n")
		b.WriteString("\n")
	}
	b.WriteString("  " + m.spinner.View() + "  " + ui.Base.Render(label) + "\n")
	return lipgloss.NewStyle().Height(h).Render(b.String())
}

// appendPreviewSection appends a live arrangement preview beneath body when
// enough vertical room remains for it; otherwise body is returned unchanged.
// The remaining room is measured from the rendered body so every screen fits
// the terminal without per-screen line accounting. Mirror layouts render a
// source box with a caption instead of positioned boxes.
func (m tuiModel) appendPreviewSection(body string, configs []layout.MonitorConfig, direction layout.Direction, mirrored bool, sourceIdx int) string {
	body = strings.TrimRight(body, "\n")
	if len(configs) == 0 {
		return body
	}
	contentH := m.height - 2
	// 3 lines of chrome around the diagram: blank, "Preview" title, blank.
	availH := contentH - lipgloss.Height(body) - 3
	previewW := m.width - 6
	if previewW < 24 {
		previewW = 24
	}
	var preview string
	if mirrored {
		preview = renderMirrorPreview(m.monitors, configs, sourceIdx, previewW, availH)
	} else {
		preview = renderDirectionPreview(m.monitors, configs, direction, previewW, availH)
	}
	if preview == "" {
		return body
	}
	return body + "\n\n  " + ui.Title.Render("Preview") + "\n\n" + ui.Indent(preview, 2)
}

// mirrorSource returns the source monitor index for the given active indexes
// when the mirror layout is in play.
func (m tuiModel) mirrorSource(activeIndexes []int) int {
	if len(activeIndexes) == 0 {
		return 0
	}
	return layout.MirrorSourceIndex(m.monitors, activeIndexes)
}

// transformLabels name the eight Hyprland transform values: 0-3 rotate
// counter-clockwise in 90° steps, 4-7 are the same rotations of a flipped
// (mirrored) image.

func (m tuiModel) layoutSelectView(h int) string {
	var b strings.Builder
	b.WriteString("\n  " + ui.Title.Render("Detected monitors") + "\n\n")
	for _, mon := range m.monitors {
		focused := "  " + ui.Dimmed.Render("○")
		if mon.Focused {
			focused = "  " + ui.Success.Render("●")
		}
		b.WriteString(fmt.Sprintf("%s  %-14s %s\n",
			focused,
			ui.Base.Render(mon.Name),
			ui.Dimmed.Render(fmt.Sprintf("%dx%d @ %sHz", mon.Width, mon.Height, ui.FormatFloat(mon.RefreshRate))),
		))
	}
	b.WriteString("\n  " + ui.Title.Render("Select a layout") + "\n\n")
	b.WriteString(m.list.View())

	body := b.String()
	// Live preview of which monitors the highlighted layout would activate.
	if item, ok := m.list.SelectedItem().(layoutListItem); ok && item.opt.ID != layout.Quit {
		indexes, err := layout.ActiveIndexes(m.monitors, item.opt.ID)
		if err != nil {
			body = strings.TrimRight(body, "\n") + "\n\n  " + ui.Dimmed.Render("Not available: "+err.Error())
		} else {
			order, direction := layout.CurrentArrangement(m.monitors, indexes)
			configs := layout.BuildConfigs(m.monitors, order, nil)
			mirrored := item.opt.ID == layout.Mirror
			body = m.appendPreviewSection(body, configs, direction, mirrored, m.mirrorSource(indexes))
		}
	}

	return lipgloss.NewStyle().Height(h).Render(body)
}

// modeSelectView shows the mode list with a live preview of the arrangement
// using the highlighted mode for the monitor being configured.

func (m tuiModel) doneView(h int) string {
	var b strings.Builder
	b.WriteString("\n")
	for _, line := range m.statusLines {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n  " + ui.Dimmed.Render("Press any key to exit.") + "\n")
	return lipgloss.NewStyle().Height(h).Render(b.String())
}

func (m tuiModel) errView(h int) string {
	msg := "An unknown error occurred."
	if m.err != nil {
		msg = m.err.Error()
	}
	boxWidth := m.width - 8
	if boxWidth < 10 {
		boxWidth = 10
	}
	box := ui.ErrorBox.Copy().Width(boxWidth).Render(
		ui.Error.Copy().Bold(true).Render("Error") + "\n\n" + ui.Base.Render(msg),
	)
	var b strings.Builder
	b.WriteString("\n\n  " + box + "\n\n")
	b.WriteString("  " + ui.Dimmed.Render("Press any key to exit.") + "\n")
	return lipgloss.NewStyle().Height(h).Render(b.String())
}
