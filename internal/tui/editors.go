package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nsumbadze/hypr-layout/internal/layout"
	"github.com/nsumbadze/hypr-layout/internal/profile"
	"github.com/nsumbadze/hypr-layout/internal/ui"
)

// ── editor screens ────────────────────────────────────────────────────────────

func (m tuiModel) updateModeSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		item, ok := m.list.SelectedItem().(modeListItem)
		if !ok {
			return m, nil
		}
		m.activeConfigs[m.editIdx].Mode = item.mode
		return m.backToReview(), nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m tuiModel) updateTransformSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		item, ok := m.list.SelectedItem().(settingListItem)
		if !ok {
			return m, nil
		}
		m.activeConfigs[m.editIdx].Transform = item.value
		return m.backToReview(), nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m tuiModel) updateVRRSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		item, ok := m.list.SelectedItem().(settingListItem)
		if !ok {
			return m, nil
		}
		m.activeConfigs[m.editIdx].VRR = item.value
		return m.backToReview(), nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m tuiModel) updateDirectionSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		item, ok := m.list.SelectedItem().(dirListItem)
		if !ok {
			return m, nil
		}
		m.direction = item.dir
		return m.backToReview(), nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// updateOrderEdit moves the highlighted monitor through the order with
// shift+↑/↓, redrawing the live preview on every move.
func (m tuiModel) updateOrderEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(km, tuiKeys.MoveUp):
		if m.editIdx > 0 {
			m.activeConfigs[m.editIdx-1], m.activeConfigs[m.editIdx] = m.activeConfigs[m.editIdx], m.activeConfigs[m.editIdx-1]
			m.editIdx--
		}
	case key.Matches(km, tuiKeys.MoveDown):
		if m.editIdx < len(m.activeConfigs)-1 {
			m.activeConfigs[m.editIdx+1], m.activeConfigs[m.editIdx] = m.activeConfigs[m.editIdx], m.activeConfigs[m.editIdx+1]
			m.editIdx++
		}
	case key.Matches(km, tuiKeys.Up):
		if m.editIdx > 0 {
			m.editIdx--
		}
	case key.Matches(km, tuiKeys.Down):
		if m.editIdx < len(m.activeConfigs)-1 {
			m.editIdx++
		}
	case key.Matches(km, tuiKeys.Enter):
		m.orderBackup = nil
		return m.backToReview(), nil
	}
	return m, nil
}

func (m tuiModel) updateConfigView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		return m.backToReview(), nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m tuiModel) updateProfileName(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		name := strings.TrimSpace(m.textInput.Value())
		if err := profile.ValidateName(name); err != nil {
			m.inputErr = err.Error()
			return m, nil
		}
		// Profiles are stored in the classic syntax whatever Hyprland reads,
		// so they stay portable and are translated again when applied.
		lines := layout.ConfLines(m.buildRules())
		meta := m.profileMetadata()
		return m, func() tea.Msg {
			dir, err := profile.DefaultDir()
			if err != nil {
				return profileSavedMsg{err: err}
			}
			return profileSavedMsg{err: profile.Save(dir, name, lines, meta)}
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m tuiModel) directionSelectView(h int) string {
	var b strings.Builder
	b.WriteString("\n  " + ui.Title.Render("Select layout direction") + "\n\n")
	b.WriteString(m.list.View())

	body := b.String()
	// Live preview: recalculate and redraw for whichever item is highlighted.
	if item, ok := m.list.SelectedItem().(dirListItem); ok {
		body = m.appendPreviewSection(body, m.activeConfigs, item.dir, false, 0)
	}

	return lipgloss.NewStyle().Height(h).Render(body)
}

func (m tuiModel) modeSelectView(h int) string {
	title := "Select mode"
	if mon, ok := m.editedMonitor(); ok {
		title = "Select mode for " + mon.Name
	}

	var b strings.Builder
	b.WriteString("\n  " + ui.Title.Render(title) + "\n\n")
	b.WriteString(m.list.View())

	body := b.String()
	if item, ok := m.list.SelectedItem().(modeListItem); ok && m.editIdx < len(m.activeConfigs) {
		configs := make([]layout.MonitorConfig, len(m.activeConfigs))
		copy(configs, m.activeConfigs)
		configs[m.editIdx].Mode = item.mode
		body = m.appendPreviewSection(body, configs, m.direction, m.mirrored, m.mirrorSource(m.activeIndexes))
	}

	return lipgloss.NewStyle().Height(h).Render(body)
}

// transformSelectView shows the rotation list with a live preview: hovering a
// 90°/270° transform visibly swaps the monitor's box proportions.
func (m tuiModel) transformSelectView(h int) string {
	title := "Select rotation"
	if mon, ok := m.editedMonitor(); ok {
		title = "Select rotation for " + mon.Name
	}

	var b strings.Builder
	b.WriteString("\n  " + ui.Title.Render(title) + "\n\n")
	b.WriteString(m.list.View())

	body := b.String()
	if item, ok := m.list.SelectedItem().(settingListItem); ok && m.editIdx < len(m.activeConfigs) {
		configs := make([]layout.MonitorConfig, len(m.activeConfigs))
		copy(configs, m.activeConfigs)
		configs[m.editIdx].Transform = item.value
		body = m.appendPreviewSection(body, configs, m.direction, m.mirrored, m.mirrorSource(m.activeIndexes))
	}

	return lipgloss.NewStyle().Height(h).Render(body)
}

// orderEditView lists the monitors in their current order with the one being
// moved marked, and redraws the arrangement preview after every move.
// vrrSelectView lists the VRR options; VRR has no spatial effect, so the
// preview simply keeps the arrangement visible for context.
func (m tuiModel) vrrSelectView(h int) string {
	title := "Select VRR"
	if mon, ok := m.editedMonitor(); ok {
		title = "Select VRR for " + mon.Name
	}

	var b strings.Builder
	b.WriteString("\n  " + ui.Title.Render(title) + "\n\n")
	b.WriteString(m.list.View())

	body := m.appendPreviewSection(b.String(), m.activeConfigs, m.direction, m.mirrored, m.mirrorSource(m.activeIndexes))
	return lipgloss.NewStyle().Height(h).Render(body)
}

func (m tuiModel) orderEditView(h int) string {
	var b strings.Builder
	b.WriteString("\n  " + ui.Title.Render("Reorder monitors") + "\n\n")
	for i, cfg := range m.activeConfigs {
		mon := m.monitors[cfg.Index]
		name := ui.Base.Render(fmt.Sprintf("%-12s", mon.Name))
		mode := ui.Dimmed.Render(layout.FormatMode(cfg.Mode))
		if i == m.editIdx {
			b.WriteString("  " + ui.Selected.Render("▸ "+fmt.Sprintf("%-12s", mon.Name)) + "  " + mode + "  " + ui.Accent.Render("↕") + "\n")
			continue
		}
		b.WriteString("    " + name + "  " + mode + "\n")
	}

	body := m.appendPreviewSection(b.String(), m.activeConfigs, m.direction, false, 0)
	return lipgloss.NewStyle().Height(h).Render(body)
}

// configView shows the exact lines that will be written to the config file.
func (m tuiModel) configView(h int) string {
	inner := m.viewport.View()
	boxWidth := m.width - 6
	if boxWidth < 10 {
		boxWidth = 10
	}
	box := ui.PreviewBox.Copy().Width(boxWidth).Render(inner)

	var b strings.Builder
	b.WriteString("\n  " + ui.Title.Render("Will be written to "+filepath.Base(m.target.Path)) + "\n\n")
	b.WriteString("  " + box + "\n")

	body := m.appendPreviewSection(b.String(), m.activeConfigs, m.direction, m.mirrored, m.mirrorSource(m.activeIndexes))
	return lipgloss.NewStyle().Height(h).Render(body)
}

func (m tuiModel) profileNameView(h int) string {
	var b strings.Builder
	b.WriteString("\n  " + ui.Title.Render("Profile name") + "\n\n")
	b.WriteString("  " + ui.Muted.Render("Letters, numbers, dashes, underscores only.") + "\n\n")
	b.WriteString("  " + m.textInput.View() + "\n")
	if m.inputErr != "" {
		b.WriteString("\n  " + ui.Error.Render(m.inputErr) + "\n")
	}
	return lipgloss.NewStyle().Height(h).Render(b.String())
}
