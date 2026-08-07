package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// runInteractive runs the full Bubble Tea wizard for the interactive flow.
// All state management (detect → layout → mode → direction → order →
// preview → save profile → apply → reload) happens inside the TUI model.
func Run() error {
	p := tea.NewProgram(newTUIModel(), tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	final, ok := finalModel.(tuiModel)
	if !ok {
		return fmt.Errorf("unexpected model type after TUI exit")
	}
	return final.err
}
