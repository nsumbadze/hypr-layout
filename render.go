package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderPreview is used by non-interactive flows (quick, apply).
func renderPreview(layoutName string, lines []string) string {
	title := styleTitle.Render("Selected: " + layoutName)
	inner := strings.Join(lines, "\n")
	box := stylePreviewBox.Copy().Width(60).Render(styleMuted.Render(inner))
	return "\n  " + title + "\n\n  " + box + "\n\n"
}

// renderProfileList is used by `hypr-layout list`.
func renderProfileList(profiles []profileSummary) string {
	if len(profiles) == 0 {
		return "\n  " + styleTitle.Render("Saved profiles") + "\n\n  " + styleDimmed.Render("No saved profiles.") + "\n\n"
	}
	var b strings.Builder
	b.WriteString("\n  " + styleTitle.Render("Saved profiles") + "\n\n")
	for _, p := range profiles {
		b.WriteString("  " + styleAccent.Render("·") + "  " + styleBase.Render(p.Name))
		if details := profileDetails(p.Meta); details != "" {
			b.WriteString("  " + styleDimmed.Render(details))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

// profileDetails builds a one-line metadata summary for the profile list.
func profileDetails(meta profileMetadata) string {
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
	b.WriteString("  " + styleDimmed.Render("1.") + "  " + styleBase.Render("Left → right") + "\n")
	b.WriteString("  " + styleDimmed.Render("2.") + "  " + styleBase.Render("Right → left") + "\n")
	b.WriteString("  " + styleDimmed.Render("3.") + "  " + styleBase.Render("Top → bottom") + "\n")
	b.WriteString("  " + styleDimmed.Render("4.") + "  " + styleBase.Render("Bottom → top") + "\n")
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

// ── direction layout preview ──────────────────────────────────────────────────

// layoutPreviewEntry holds the visual position and display data for one monitor
// in the ASCII layout diagram.
type layoutPreviewEntry struct {
	name   string
	mode   monitorMode
	width  int // effective pixel width after transform
	height int // effective pixel height after transform
	pixelX int
	pixelY int
}

// renderDirectionPreview returns a proportionally-scaled ASCII art diagram
// showing how the active monitors will be arranged for the given direction.
// It is called live while the user navigates the direction list in the TUI.
// availHeight caps the total diagram height so it fits the terminal; when
// there is not enough room for even minimal boxes, an empty string is
// returned and the caller skips the preview section.
func renderDirectionPreview(monitors []monitor, configs []activeMonitorConfig, direction layoutDirection, availWidth, availHeight int) string {
	if len(configs) == 0 {
		return ""
	}

	entries := buildLayoutEntries(monitors, configs, direction)

	// Normalise so the top-left corner of the bounding box is (0, 0).
	minX, minY := entries[0].pixelX, entries[0].pixelY
	for _, e := range entries {
		if e.pixelX < minX {
			minX = e.pixelX
		}
		if e.pixelY < minY {
			minY = e.pixelY
		}
	}
	for i := range entries {
		entries[i].pixelX -= minX
		entries[i].pixelY -= minY
	}

	isHorizontal := direction == leftToRight || direction == rightToLeft

	// Sort by visual position so boxes appear left-to-right or top-to-bottom.
	sort.Slice(entries, func(i, j int) bool {
		if isHorizontal {
			return entries[i].pixelX < entries[j].pixelX
		}
		return entries[i].pixelY < entries[j].pixelY
	})

	if isHorizontal {
		return renderHorizPreview(entries, availWidth, availHeight)
	}
	return renderVertPreview(entries, availWidth, availHeight)
}

// buildLayoutEntries computes pixel positions using the same arithmetic as
// renderPositionedConfigLines so the preview always matches the real output.
func buildLayoutEntries(monitors []monitor, configs []activeMonitorConfig, direction layoutDirection) []layoutPreviewEntry {
	entries := make([]layoutPreviewEntry, len(configs))
	posX, posY := 0, 0

	for i, cfg := range configs {
		mon := monitors[cfg.Index]
		effWidth, effHeight := effectiveModeSize(cfg.Mode, cfg.Transform)

		if i > 0 {
			switch direction {
			case rightToLeft:
				posX -= effWidth
			case bottomToTop:
				posY -= effHeight
			}
		}

		entries[i] = layoutPreviewEntry{
			name:   mon.Name,
			mode:   cfg.Mode,
			width:  effWidth,
			height: effHeight,
			pixelX: posX,
			pixelY: posY,
		}

		switch direction {
		case leftToRight:
			posX += effWidth
		case topToBottom:
			posY += effHeight
		}
	}
	return entries
}

// boxOverhead is the total horizontal chars consumed by border (1+1) and
// padding (1+1) in each preview box.
const boxOverhead = 4

// minPreviewBoxH is the shortest useful box: two border lines plus the
// monitor name.
const minPreviewBoxH = 3

func renderHorizPreview(entries []layoutPreviewEntry, availWidth, availHeight int) string {
	boxH := 5
	if boxH > availHeight {
		boxH = availHeight
	}
	if boxH < minPreviewBoxH {
		return ""
	}

	n := len(entries)
	totalPxW := 0
	for _, e := range entries {
		totalPxW += e.width
	}
	if totalPxW == 0 {
		return ""
	}

	// Distribute available chars proportionally; reserve 1 char gap between boxes.
	usableW := availWidth - (n - 1)
	minPerBox := boxOverhead + 6
	if usableW < n*minPerBox {
		usableW = n * minPerBox
	}

	boxes := make([]string, n)
	for i, e := range entries {
		totalBoxW := (e.width * usableW) / totalPxW
		if totalBoxW < minPerBox {
			totalBoxW = minPerBox
		}
		// Right margin of 1 between boxes (except last) creates the visual gap
		// without needing an explicit spacer element.
		marginRight := 0
		if i < n-1 {
			marginRight = 1
		}
		boxes[i] = previewBox(e.name, e.mode, totalBoxW-boxOverhead, boxH, marginRight)
	}

	// JoinHorizontal places multi-line boxes side by side correctly.
	return lipgloss.JoinHorizontal(lipgloss.Top, boxes...)
}

func renderVertPreview(entries []layoutPreviewEntry, availWidth, availHeight int) string {
	n := len(entries)
	if n == 0 || availHeight < n*minPreviewBoxH {
		return ""
	}

	maxTotalH := availHeight
	if maxTotalH > 18 {
		maxTotalH = 18
	}

	totalPxH := 0
	maxPxW := 0
	for _, e := range entries {
		totalPxH += e.height
		if e.width > maxPxW {
			maxPxW = e.width
		}
	}
	if totalPxH == 0 || maxPxW == 0 {
		return ""
	}

	maxInnerW := availWidth - boxOverhead
	if maxInnerW < 12 {
		maxInnerW = 12
	}

	// Scale heights proportionally to monitor height, with a minimum per box.
	heights := make([]int, n)
	total := 0
	for i, e := range entries {
		heights[i] = (e.height * maxTotalH) / totalPxH
		if heights[i] < minPreviewBoxH {
			heights[i] = minPreviewBoxH
		}
		total += heights[i]
	}
	// Minimum clamping can push the total past the budget; shrink the tallest
	// boxes until everything fits (guaranteed to terminate because
	// availHeight >= n*minPreviewBoxH).
	for total > availHeight {
		tallest := 0
		for i, h := range heights {
			if h > heights[tallest] {
				tallest = i
			}
		}
		if heights[tallest] <= minPreviewBoxH {
			break
		}
		heights[tallest]--
		total--
	}

	boxes := make([]string, n)
	for i, e := range entries {
		// Scale width relative to the widest monitor in this layout.
		innerW := (e.width * maxInnerW) / maxPxW
		if innerW < 12 {
			innerW = 12
		}
		boxes[i] = previewBox(e.name, e.mode, innerW, heights[i], 0)
	}

	return strings.Join(boxes, "\n")
}

// previewBox renders a single labelled monitor box for the direction preview.
// innerW is the content width (excluding border/padding), totalH is the full
// rendered height (including border lines), marginRight adds right margin.
func previewBox(name string, mode monitorMode, innerW, totalH, marginRight int) string {
	if innerW < 4 {
		innerW = 4
	}

	trunc := func(s string, max int) string {
		if len(s) <= max {
			return s
		}
		if max <= 1 {
			return "…"
		}
		return s[:max-1] + "…"
	}

	// Interior height = totalH minus top and bottom border chars.
	innerH := totalH - 2
	if innerH < 1 {
		innerH = 1
	}

	contentLines := make([]string, innerH)
	contentLines[0] = styleBase.Render(trunc(name, innerW))
	if innerH >= 2 {
		contentLines[1] = styleDimmed.Render(trunc(formatMonitorMode(mode), innerW))
	}
	// Remaining lines stay empty (padding).

	content := strings.Join(contentLines, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1).
		Width(innerW).
		MarginRight(marginRight).
		Render(content)
}

// indentBlock adds n spaces to the start of every non-empty line in s.
// Used to align multi-line preview blocks with the rest of the TUI body.
func indentBlock(s string, n int) string {
	prefix := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = prefix + l
		}
	}
	return strings.Join(lines, "\n")
}

// ── shared formatting ─────────────────────────────────────────────────────────

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
