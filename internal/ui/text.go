package ui

import (
	"strconv"
	"strings"
)

// Indent adds n spaces to the start of every non-empty line in s.
// Used to align multi-line preview blocks with the rest of the TUI body.
func Indent(s string, n int) string {
	prefix := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = prefix + l
		}
	}
	return strings.Join(lines, "\n")
}

func FormatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
