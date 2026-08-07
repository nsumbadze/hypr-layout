package cli

import (
	"fmt"
	"strconv"

	"github.com/nsumbadze/hypr-layout/internal/layout"
)

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

// keepCurrentSetting marks a transform/vrr option as "not overridden": each
// monitor keeps whatever value Hyprland currently reports for it.
const keepCurrentSetting = -1

type quickOptions struct {
	commandOptions
	Direction layout.Direction
	Order     string
	Mode      layout.Strategy
	Transform int
	VRR       int
}

func parseQuickOptions(args []string) (quickOptions, error) {
	options := quickOptions{
		Direction: layout.LeftToRight,
		Mode:      layout.StrategyBest,
		Transform: keepCurrentSetting,
		VRR:       keepCurrentSetting,
	}

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
			direction, err := layout.ParseDirection(value)
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
		case "--mode":
			value, next, err := flagValue(args, i)
			if err != nil {
				return quickOptions{}, err
			}
			strategy, err := layout.ParseStrategy(value)
			if err != nil {
				return quickOptions{}, err
			}
			options.Mode = strategy
			i = next
		case "--transform":
			value, next, err := flagValue(args, i)
			if err != nil {
				return quickOptions{}, err
			}
			transform, err := parseBoundedInt(value, 0, 7, "--transform")
			if err != nil {
				return quickOptions{}, err
			}
			options.Transform = transform
			i = next
		case "--vrr":
			value, next, err := flagValue(args, i)
			if err != nil {
				return quickOptions{}, err
			}
			vrr, err := parseBoundedInt(value, 0, 2, "--vrr")
			if err != nil {
				return quickOptions{}, err
			}
			options.VRR = vrr
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

func parseBoundedInt(value string, min, max int, flag string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < min || parsed > max {
		return 0, fmt.Errorf("%s must be a number between %d and %d", flag, min, max)
	}

	return parsed, nil
}

// applyDisplayOverrides applies global --transform/--vrr overrides to every
// active monitor; keepCurrentSetting leaves the detected value untouched.
func applyDisplayOverrides(configs []layout.MonitorConfig, transform, vrr int) {
	for i := range configs {
		if transform != keepCurrentSetting {
			configs[i].Transform = transform
		}
		if vrr != keepCurrentSetting {
			configs[i].VRR = vrr
		}
	}
}

func quickPresetLayoutID(preset string) (int, error) {
	switch preset {
	case "laptop":
		return layout.LaptopOnly, nil
	case "external":
		return layout.ExternalOnly, nil
	case "dual":
		return layout.DualHorizontal, nil
	case "triple":
		return layout.TripleHorizontal, nil
	case "mirror":
		return layout.Mirror, nil
	default:
		return 0, fmt.Errorf("unknown preset: %s", preset)
	}
}
