package main

import (
	"errors"
	"fmt"
	"os"
)

func exitWithFriendlyMonitorError(err error) {
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "Could not read Hyprland monitors. Is Hyprland running?")
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Could not read Hyprland monitors. Is Hyprland running?")
	os.Exit(1)
}
