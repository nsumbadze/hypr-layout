package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type monitor struct {
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

func parseMonitors(data []byte) ([]monitor, error) {
	var monitors []monitor
	if err := json.Unmarshal(data, &monitors); err != nil {
		return nil, fmt.Errorf("decode monitor JSON: %w", err)
	}

	return monitors, nil
}
