package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var monitorModePattern = regexp.MustCompile(`^(\d+)x(\d+)@([0-9]+(?:\.[0-9]+)?)Hz$`)

type monitorMode struct {
	Width       int
	Height      int
	RefreshRate float64
}

func promptMonitorModes(r io.Reader, w io.Writer, monitors []monitor, activeIndexes []int) (map[int]monitorMode, error) {
	reader := bufio.NewReader(r)
	selected := make(map[int]monitorMode, len(activeIndexes))

	for _, idx := range activeIndexes {
		mon := monitors[idx]
		current := currentMonitorMode(mon)
		modes := availableMonitorModes(mon)

		if len(modes) == 0 {
			selected[idx] = current
			continue
		}

		for {
			fmt.Fprint(w, renderMonitorModes(mon, current, modes))

			input, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					choice := strings.TrimSpace(input)
					if choice == "" {
						selected[idx] = current
						break
					}
				}

				return nil, err
			}

			mode, parseErr := parseMonitorModeSelection(input, modes, current)
			if parseErr != nil {
				fmt.Fprintf(w, "Invalid selection: %v\n\n", parseErr)
				continue
			}

			selected[idx] = mode
			break
		}
	}

	return selected, nil
}

func parseMonitorModeSelection(input string, modes []monitorMode, current monitorMode) (monitorMode, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return current, nil
	}

	choice, err := strconv.Atoi(value)
	if err != nil {
		return monitorMode{}, fmt.Errorf("enter a valid number")
	}

	if choice < 1 || choice > len(modes) {
		return monitorMode{}, fmt.Errorf("enter a number between 1 and %d", len(modes))
	}

	return modes[choice-1], nil
}

func availableMonitorModes(mon monitor) []monitorMode {
	seen := make(map[string]struct{}, len(mon.AvailableModes))
	modes := make([]monitorMode, 0, len(mon.AvailableModes))

	for _, raw := range mon.AvailableModes {
		mode, err := parseMonitorMode(raw)
		if err != nil {
			continue
		}

		key := formatMonitorMode(mode)
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		modes = append(modes, mode)
	}

	return modes
}

func currentMonitorMode(mon monitor) monitorMode {
	return monitorMode{
		Width:       mon.Width,
		Height:      mon.Height,
		RefreshRate: mon.RefreshRate,
	}
}

func parseMonitorMode(value string) (monitorMode, error) {
	matches := monitorModePattern.FindStringSubmatch(strings.TrimSpace(value))
	if matches == nil {
		return monitorMode{}, fmt.Errorf("invalid monitor mode: %q", value)
	}

	width, err := strconv.Atoi(matches[1])
	if err != nil {
		return monitorMode{}, fmt.Errorf("parse width: %w", err)
	}

	height, err := strconv.Atoi(matches[2])
	if err != nil {
		return monitorMode{}, fmt.Errorf("parse height: %w", err)
	}

	refreshRate, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return monitorMode{}, fmt.Errorf("parse refresh rate: %w", err)
	}

	return monitorMode{
		Width:       width,
		Height:      height,
		RefreshRate: refreshRate,
	}, nil
}

func formatMonitorMode(mode monitorMode) string {
	return fmt.Sprintf("%dx%d@%s", mode.Width, mode.Height, formatFloat(mode.RefreshRate))
}
