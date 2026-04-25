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
