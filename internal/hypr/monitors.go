package hypr

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

type Monitor struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Width          int      `json:"width"`
	Height         int      `json:"height"`
	RefreshRate    float64  `json:"refreshRate"`
	X              int      `json:"x"`
	Y              int      `json:"y"`
	Scale          float64  `json:"scale"`
	Transform      int      `json:"transform"`
	VRR            bool     `json:"vrr"`
	Focused        bool     `json:"focused"`
	AvailableModes []string `json:"availableModes"`
}

func readMonitorsJSON(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "hyprctl", "monitors", "-j")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("run hyprctl monitors -j: %w", err)
	}

	return output, nil
}

func parseMonitors(data []byte) ([]Monitor, error) {
	var monitors []Monitor
	if err := json.Unmarshal(data, &monitors); err != nil {
		return nil, fmt.Errorf("decode monitor JSON: %w", err)
	}

	return monitors, nil
}

func Detect() ([]Monitor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	output, err := readMonitorsJSON(ctx)
	if err != nil {
		return nil, friendlyMonitorError(err)
	}
	monitors, err := parseMonitors(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Hyprland monitor data: %w", err)
	}
	return monitors, nil
}

func friendlyMonitorError(_ error) error {
	return fmt.Errorf("Could not read Hyprland monitors. Is Hyprland running?")
}
