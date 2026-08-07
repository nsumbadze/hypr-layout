package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
	"github.com/nsumbadze/hypr-layout/internal/layout"
	"github.com/nsumbadze/hypr-layout/internal/ui"
)

// in the ASCII layout diagram.
type layoutPreviewEntry struct {
	name   string
	mode   layout.Mode
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
func renderDirectionPreview(monitors []hypr.Monitor, configs []layout.MonitorConfig, direction layout.Direction, availWidth, availHeight int) string {
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

	isHorizontal := direction == layout.LeftToRight || direction == layout.RightToLeft

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
// layout.PositionedLines so the preview always matches the real output.
func buildLayoutEntries(monitors []hypr.Monitor, configs []layout.MonitorConfig, direction layout.Direction) []layoutPreviewEntry {
	entries := make([]layoutPreviewEntry, len(configs))
	posX, posY := 0, 0

	for i, cfg := range configs {
		mon := monitors[cfg.Index]
		effWidth, effHeight := layout.EffectiveSize(cfg.Mode, cfg.Transform)

		if i > 0 {
			switch direction {
			case layout.RightToLeft:
				posX -= effWidth
			case layout.BottomToTop:
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
		case layout.LeftToRight:
			posX += effWidth
		case layout.TopToBottom:
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

// renderMirrorPreview draws the mirror source monitor as a single box with a
// caption listing the monitors that mirror it. Returns "" when there is not
// enough vertical room.
func renderMirrorPreview(monitors []hypr.Monitor, configs []layout.MonitorConfig, sourceIdx int, availWidth, availHeight int) string {
	if availHeight < minPreviewBoxH+1 {
		return ""
	}

	var source *layout.MonitorConfig
	mirrors := make([]string, 0, len(configs))
	for i := range configs {
		if configs[i].Index == sourceIdx {
			source = &configs[i]
			continue
		}
		mirrors = append(mirrors, monitors[configs[i].Index].Name)
	}
	if source == nil {
		return ""
	}

	boxH := 5
	if boxH > availHeight-1 {
		boxH = availHeight - 1
	}
	innerW := availWidth - boxOverhead
	if innerW > 40 {
		innerW = 40
	}
	if innerW < 12 {
		innerW = 12
	}

	box := previewBox(monitors[sourceIdx].Name, source.Mode, innerW, boxH, 0)
	if len(mirrors) == 0 {
		return box
	}
	return box + "\n" + ui.Dimmed.Render("mirrored by "+strings.Join(mirrors, ", "))
}

// previewBox renders a single labelled monitor box for the direction preview.
// innerW is the content width (excluding border/padding), totalH is the full
// rendered height (including border lines), marginRight adds right margin.
func previewBox(name string, mode layout.Mode, innerW, totalH, marginRight int) string {
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
	contentLines[0] = ui.Base.Render(trunc(name, innerW))
	if innerH >= 2 {
		contentLines[1] = ui.Dimmed.Render(trunc(layout.FormatMode(mode), innerW))
	}
	// Remaining lines stay empty (padding).

	content := strings.Join(contentLines, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorBorder).
		Padding(0, 1).
		Width(innerW).
		MarginRight(marginRight).
		Render(content)
}

// ── shared formatting ─────────────────────────────────────────────────────────
