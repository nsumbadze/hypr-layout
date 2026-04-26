package main

import (
	"fmt"
	"strconv"
	"strings"
)

func renderMonitors(monitors []monitor) string {
	if len(monitors) == 0 {
		return "Detected monitors:\n\nNo monitors detected.\n"
	}

	var b strings.Builder
	b.WriteString("Detected monitors:\n\n")

	for i, mon := range monitors {
		fmt.Fprintf(&b, "%d. %s\n", i+1, mon.Name)
		fmt.Fprintf(&b, "   Description: %s\n", fallbackDescription(mon.Description))
		fmt.Fprintf(&b, "   Resolution: %dx%d\n", mon.Width, mon.Height)
		fmt.Fprintf(&b, "   Refresh: %sHz\n", formatFloat(mon.RefreshRate))
		fmt.Fprintf(&b, "   Position: %dx%d\n", mon.X, mon.Y)
		fmt.Fprintf(&b, "   Scale: %s\n", formatFloat(mon.Scale))
		fmt.Fprintf(&b, "   Focused: %s\n", yesNo(mon.Focused))

		if i < len(monitors)-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

func renderLayoutOptions(options []layoutOption) string {
	var b strings.Builder
	b.WriteString("\nLayout options:\n\n")

	for _, option := range options {
		fmt.Fprintf(&b, "%d. %s\n", option.ID, option.Name)
	}

	b.WriteString("\nSelect a layout: ")

	return b.String()
}

func renderPreview(layoutName string, lines []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nSelected: %s\n\n", layoutName)
	b.WriteString("Preview config:\n")

	for _, line := range lines {
		fmt.Fprintf(&b, "%s\n", line)
	}

	return b.String()
}

func renderProfileList(profiles []string) string {
	if len(profiles) == 0 {
		return "Saved profiles:\n\nNo saved profiles.\n"
	}

	var b strings.Builder
	b.WriteString("Saved profiles:\n\n")
	for _, profile := range profiles {
		fmt.Fprintf(&b, "%s\n", profile)
	}

	return b.String()
}

func renderMonitorModes(mon monitor, current monitorMode, modes []monitorMode) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nModes for %s:\n", mon.Name)
	fmt.Fprintf(&b, "Current: %s\n\n", formatMonitorMode(current))

	for i, mode := range modes {
		fmt.Fprintf(&b, "%d. %s\n", i+1, formatMonitorMode(mode))
	}

	b.WriteString("\nSelect a mode (Enter keeps current): ")
	return b.String()
}

func renderLayoutDirections() string {
	var b strings.Builder
	b.WriteString("\nLayout direction:\n\n")
	b.WriteString("1. Horizontal (left -> right)\n")
	b.WriteString("2. Vertical (top -> bottom)\n")
	b.WriteString("\nSelect layout direction: ")
	return b.String()
}

func renderMonitorOrderPrompt(monitors []monitor, activeConfigs []activeMonitorConfig) string {
	var b strings.Builder
	b.WriteString("\nActive monitors:\n\n")

	for i, config := range activeConfigs {
		mon := monitors[config.Index]
		fmt.Fprintf(&b, "%d. %s (%s)\n", i+1, mon.Name, formatMonitorMode(config.Mode))
	}

	b.WriteString("\nEnter monitor order by indices (e.g. \"2 1 3\"): ")
	return b.String()
}

func fallbackDescription(description string) string {
	if description == "" {
		return "Unknown"
	}

	return description
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
