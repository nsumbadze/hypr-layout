package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type layoutDirection string

const (
	leftToRight layoutDirection = "left-to-right"
	rightToLeft layoutDirection = "right-to-left"
	topToBottom layoutDirection = "top-to-bottom"
	bottomToTop layoutDirection = "bottom-to-top"
)

type activeMonitorConfig struct {
	Index int
	Mode  monitorMode
}

func promptLayoutDirection(r io.Reader, w io.Writer) (layoutDirection, error) {
	reader := bufio.NewReader(r)

	for {
		fmt.Fprint(w, renderLayoutDirections())

		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && strings.TrimSpace(input) != "" {
				return parseLayoutDirection(input)
			}

			return "", err
		}

		direction, parseErr := parseLayoutDirection(input)
		if parseErr != nil {
			fmt.Fprintf(w, "Invalid selection: %v\n\n", parseErr)
			continue
		}

		return direction, nil
	}
}

func parseLayoutDirection(input string) (layoutDirection, error) {
	switch strings.TrimSpace(input) {
	case "1":
		return leftToRight, nil
	case "2":
		return rightToLeft, nil
	case "3":
		return topToBottom, nil
	case "4":
		return bottomToTop, nil
	default:
		return "", fmt.Errorf("enter a number between 1 and 4")
	}
}

func promptMonitorOrder(r io.Reader, w io.Writer, monitors []monitor, activeConfigs []activeMonitorConfig) ([]activeMonitorConfig, error) {
	reader := bufio.NewReader(r)

	for {
		fmt.Fprint(w, renderMonitorOrderPrompt(monitors, activeConfigs))

		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && strings.TrimSpace(input) != "" {
				return reorderActiveConfigs(strings.TrimSpace(input), activeConfigs)
			}

			return nil, err
		}

		ordered, parseErr := reorderActiveConfigs(strings.TrimSpace(input), activeConfigs)
		if parseErr != nil {
			fmt.Fprintf(w, "Invalid selection: %v\n\n", parseErr)
			continue
		}

		return ordered, nil
	}
}

func reorderActiveConfigs(input string, activeConfigs []activeMonitorConfig) ([]activeMonitorConfig, error) {
	fields := strings.Fields(input)
	if len(fields) != len(activeConfigs) {
		return nil, fmt.Errorf("enter exactly %d indices", len(activeConfigs))
	}

	ordered := make([]activeMonitorConfig, 0, len(activeConfigs))
	seen := make(map[int]struct{}, len(activeConfigs))

	for _, field := range fields {
		position, err := strconv.Atoi(field)
		if err != nil {
			return nil, fmt.Errorf("enter indices separated by spaces")
		}

		if position < 1 || position > len(activeConfigs) {
			return nil, fmt.Errorf("indices must be between 1 and %d", len(activeConfigs))
		}

		if _, ok := seen[position]; ok {
			return nil, fmt.Errorf("duplicate indices are not allowed")
		}

		seen[position] = struct{}{}
		ordered = append(ordered, activeConfigs[position-1])
	}

	return ordered, nil
}

func buildActiveMonitorConfigs(monitors []monitor, activeIndexes []int, selectedModes map[int]monitorMode) []activeMonitorConfig {
	configs := make([]activeMonitorConfig, 0, len(activeIndexes))

	for _, idx := range activeIndexes {
		mode := currentMonitorMode(monitors[idx])
		if selectedModes != nil {
			if selectedMode, ok := selectedModes[idx]; ok {
				mode = selectedMode
			}
		}

		configs = append(configs, activeMonitorConfig{
			Index: idx,
			Mode:  mode,
		})
	}

	return configs
}

// renderPositionedConfigLines generates monitor config lines for the ordered
// active monitors followed by disabled monitors.
//
// Coordinate rules (Hyprland: Y increases downward):
//
//	left-to-right : each monitor shifts right by the previous monitor's width.
//	right-to-left : each monitor shifts left by its own width (produces negative X).
//	top-to-bottom : each monitor shifts down by the previous monitor's height.
//	bottom-to-top : each monitor shifts up by its own height (produces negative Y).
func renderPositionedConfigLines(monitors []monitor, activeConfigs []activeMonitorConfig, direction layoutDirection) []string {
	lines := make([]string, 0, len(monitors))
	activeSet := make(map[int]struct{}, len(activeConfigs))
	positionX := 0
	positionY := 0

	for i, config := range activeConfigs {
		mon := monitors[config.Index]
		activeSet[config.Index] = struct{}{}

		// right-to-left and bottom-to-top subtract the current monitor's
		// dimension before placing so that monitor[0] lands at 0x0 and
		// every subsequent monitor accumulates a negative offset.
		if i > 0 {
			switch direction {
			case rightToLeft:
				positionX -= config.Mode.Width
			case bottomToTop:
				positionY -= config.Mode.Height
			}
		}

		lines = append(lines, fmt.Sprintf(
			"monitor = %s, %s, %dx%d, %s",
			mon.Name,
			formatMonitorMode(config.Mode),
			positionX,
			positionY,
			formatFloat(mon.Scale),
		))

		switch direction {
		case leftToRight:
			positionX += config.Mode.Width
		case topToBottom:
			positionY += config.Mode.Height
		}
	}

	for idx, mon := range monitors {
		if _, ok := activeSet[idx]; ok {
			continue
		}

		lines = append(lines, fmt.Sprintf("monitor = %s, disable", mon.Name))
	}

	return lines
}
