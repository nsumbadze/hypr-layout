package main

import "fmt"

type commandOptions struct {
	AutoYes  bool
	NoReload bool
}

func parseCommandOptions(args []string) (commandOptions, error) {
	var options commandOptions

	for _, arg := range args {
		switch arg {
		case "--yes":
			options.AutoYes = true
		case "--no-reload":
			options.NoReload = true
		default:
			return commandOptions{}, fmt.Errorf("unknown flag: %s", arg)
		}
	}

	return options, nil
}

type quickOptions struct {
	commandOptions
	Direction layoutDirection
	Order     string
}

func parseQuickOptions(args []string) (quickOptions, error) {
	options := quickOptions{Direction: leftToRight}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--yes":
			options.AutoYes = true
		case "--no-reload":
			options.NoReload = true
		case "--direction":
			value, next, err := flagValue(args, i)
			if err != nil {
				return quickOptions{}, err
			}
			direction, err := parseDirectionName(value)
			if err != nil {
				return quickOptions{}, err
			}
			options.Direction = direction
			i = next
		case "--order":
			value, next, err := flagValue(args, i)
			if err != nil {
				return quickOptions{}, err
			}
			options.Order = value
			i = next
		default:
			return quickOptions{}, fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	return options, nil
}

func flagValue(args []string, i int) (string, int, error) {
	if i+1 >= len(args) {
		return "", i, fmt.Errorf("%s requires a value", args[i])
	}

	return args[i+1], i + 1, nil
}

func quickPresetLayoutID(preset string) (int, error) {
	switch preset {
	case "laptop":
		return 1, nil
	case "external":
		return 2, nil
	case "dual":
		return 3, nil
	case "triple":
		return 4, nil
	default:
		return 0, fmt.Errorf("unknown preset: %s", preset)
	}
}

func autoSelectMonitorModes(monitors []monitor, activeIndexes []int) map[int]monitorMode {
	selected := make(map[int]monitorMode, len(activeIndexes))

	for _, idx := range activeIndexes {
		selected[idx] = bestMonitorMode(monitors[idx])
	}

	return selected
}

func bestMonitorMode(mon monitor) monitorMode {
	current := currentMonitorMode(mon)
	best := current
	modes := availableMonitorModes(mon)

	if len(modes) == 0 {
		return current
	}

	best = modes[0]
	for _, mode := range modes[1:] {
		if compareMonitorModes(mode, best) > 0 {
			best = mode
		}
	}

	return best
}

func compareMonitorModes(a, b monitorMode) int {
	switch {
	case a.RefreshRate > b.RefreshRate:
		return 1
	case a.RefreshRate < b.RefreshRate:
		return -1
	}

	aPixels := a.Width * a.Height
	bPixels := b.Width * b.Height
	switch {
	case aPixels > bPixels:
		return 1
	case aPixels < bPixels:
		return -1
	}

	return 0
}
