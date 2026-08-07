package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nsumbadze/hypr-layout/internal/hypr"
	"github.com/nsumbadze/hypr-layout/internal/layout"
	"github.com/nsumbadze/hypr-layout/internal/monconf"
	"github.com/nsumbadze/hypr-layout/internal/ui"
)

// ── update ────────────────────────────────────────────────────────────────────

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.resizeComponents()
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, tuiKeys.Quit) && m.state != tuiApplying && m.state != tuiReloading {
			return m, tea.Quit
		}
		if key.Matches(msg, tuiKeys.Back) && backAllowed(m.state) {
			return m.goBack()
		}

	case spinner.TickMsg:
		if m.state == tuiDetecting || m.state == tuiApplying || m.state == tuiReloading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case monDetectedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = tuiErr
			return m, nil
		}
		if len(msg.monitors) == 0 {
			m.err = fmt.Errorf("no monitors detected")
			m.state = tuiErr
			return m, nil
		}
		m.monitors = msg.monitors
		m.state = tuiLayoutSelect
		m.list = m.makeLayoutList()
		return m, nil

	case configWrittenMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = tuiErr
			return m, nil
		}
		m.applyResult = msg.result
		m.statusLines = append(m.statusLines,
			ui.Success.Render("✓")+"  Config written to "+ui.Muted.Render(msg.result.ConfigPath),
		)
		if msg.result.BackupPath != "" {
			m.statusLines = append(m.statusLines,
				ui.Dimmed.Render("  Backup: "+msg.result.BackupPath),
			)
		}
		return m.startReload()

	case reloadDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = tuiErr
			return m, nil
		}
		m.statusLines = append(m.statusLines,
			ui.Success.Render("✓")+"  Hyprland reloaded.",
		)
		m.state = tuiDone
		return m, nil

	case profileSavedMsg:
		// A failed save keeps the name screen open so the name can be fixed;
		// a successful one drops straight back to the hub.
		if msg.err != nil {
			m.inputErr = "Failed to save profile: " + msg.err.Error()
			return m, nil
		}
		m.textInput.Blur()
		m.statusLines = append(m.statusLines,
			ui.Success.Render("✓")+"  Profile saved.",
		)
		m.state = tuiReview
		return m, nil
	}

	switch m.state {
	case tuiLayoutSelect:
		return m.updateLayoutSelect(msg)
	case tuiReview:
		return m.updateReview(msg)
	case tuiModeSelect:
		return m.updateModeSelect(msg)
	case tuiTransformSelect:
		return m.updateTransformSelect(msg)
	case tuiVRRSelect:
		return m.updateVRRSelect(msg)
	case tuiDirectionSelect:
		return m.updateDirectionSelect(msg)
	case tuiOrderEdit:
		return m.updateOrderEdit(msg)
	case tuiConfigView:
		return m.updateConfigView(msg)
	case tuiProfileName:
		return m.updateProfileName(msg)
	case tuiDone, tuiErr:
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, tea.Quit
		}
	}

	return m, nil
}

// ── state update helpers ──────────────────────────────────────────────────────

func (m tuiModel) updateLayoutSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, tuiKeys.Enter) {
		item, ok := m.list.SelectedItem().(layoutListItem)
		if !ok {
			return m, nil
		}
		if item.opt.ID == layout.Quit {
			return m, tea.Quit
		}
		indexes, err := layout.ActiveIndexes(m.monitors, item.opt.ID)
		if err != nil {
			m.err = err
			m.state = tuiErr
			return m, nil
		}
		m.activeIndexes = indexes
		m.mirrored = item.opt.ID == layout.Mirror
		m.direction = layout.LeftToRight
		// nil modes means every setting starts at what the monitor is running
		// now, so the hub opens showing the current setup rather than a
		// proposed new one.
		m.activeConfigs = layout.BuildConfigs(m.monitors, indexes, nil)
		m.reviewCursor = firstSelectableRow(m.reviewRows())
		m.state = tuiReview
		m = m.resizeComponents()
		return m, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// ── apply ─────────────────────────────────────────────────────────────────────

// startApply writes the config and reloads Hyprland. There is no confirmation
// step: the previous file is backed up first and a failed reload rolls back
// automatically.
func (m tuiModel) startApply() (tea.Model, tea.Cmd) {
	m.configLines = m.buildConfigLines()
	m.state = tuiApplying
	lines := m.configLines
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			path, err := monconf.DefaultPath()
			if err != nil {
				return configWrittenMsg{err: err}
			}
			result, err := monconf.Apply(path, lines, time.Now())
			return configWrittenMsg{result: result, err: err}
		},
	)
}

func (m tuiModel) startReload() (tea.Model, tea.Cmd) {
	m.state = tuiReloading
	result := m.applyResult
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := hypr.Reload(ctx, hypr.SystemRunner); err != nil {
				if rbErr := monconf.Rollback(result); rbErr != nil {
					return reloadDoneMsg{err: fmt.Errorf("reload failed: %v; rollback failed: %w", err, rbErr)}
				}
				if result.BackupPath != "" {
					return reloadDoneMsg{err: fmt.Errorf("reload failed: %v; restored backup from %s", err, result.BackupPath)}
				}
				return reloadDoneMsg{err: fmt.Errorf("reload failed: %v; removed newly created config", err)}
			}
			return reloadDoneMsg{}
		},
	)
}

// ── back navigation ───────────────────────────────────────────────────────────

// backAllowed returns true for states where pressing Escape navigates back.
// Editor screens return to the review hub and the hub returns to the layout
// list; states where a side-effect is in flight (writing, reloading) or where
// there is nothing behind them (detecting, done, error) do not allow it.
func backAllowed(s tuiState) bool {
	return s == tuiReview || isEditorState(s)
}

func (m tuiModel) goBack() (tuiModel, tea.Cmd) {
	if m.state == tuiReview {
		m.state = tuiLayoutSelect
		m.list = m.makeLayoutList()
		return m, nil
	}
	// Reordering mutates the live order as it goes, so cancelling has to put
	// the pre-edit order back.
	if m.state == tuiOrderEdit && m.orderBackup != nil {
		m.activeConfigs = m.orderBackup
		m.orderBackup = nil
	}
	return m.backToReview(), nil
}
