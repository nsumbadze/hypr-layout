// Package cli is the command line: subcommand dispatch and the
// non-interactive flows for quick presets and saved profiles.
package cli

import (
	"fmt"
	"os"

	"github.com/nsumbadze/hypr-layout/internal/tui"
)

// Main runs the command line and returns the process exit code, reporting any
// failure on stderr.
func Main(args []string) int {
	if err := run(args); err != nil {
		fmt.Fprintln(os.Stderr, renderInlineError(err.Error()))
		return 1
	}
	return 0
}

func run(args []string) error {
	switch {
	case len(args) == 0:
		return tui.Run()
	case len(args) == 1 && args[0] == "list":
		return runListProfiles()
	case len(args) >= 2 && args[0] == "apply":
		return runApplyProfile(args[1], args[2:])
	case len(args) >= 2 && args[0] == "quick":
		return runQuickPreset(args[1], args[2:])
	case len(args) >= 1 && len(args) <= 2 && args[0] == "export":
		exportPath := ""
		if len(args) == 2 {
			exportPath = args[1]
		}
		return runExportProfiles(exportPath)
	case len(args) >= 2 && args[0] == "import":
		return runImportProfiles(args[1], args[2:])
	default:
		return fmt.Errorf("usage: hypr-layout [list | apply <profile-name> [--yes] [--no-reload] | quick <preset> [--yes] [--no-reload] [--direction <dir>] [--order <indices>] [--mode <strategy>] [--transform <0-7>] [--vrr <0-2>] | export [file] | import <file> [--force]]")
	}
}
