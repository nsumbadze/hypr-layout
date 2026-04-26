package main

import (
	"fmt"
)

func friendlyMonitorError(_ error) error {
	return fmt.Errorf("Could not read Hyprland monitors. Is Hyprland running?")
}
