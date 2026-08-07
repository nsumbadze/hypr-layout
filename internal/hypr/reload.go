package hypr

import (
	"context"
	"fmt"
	"os/exec"
)

type CommandRunner func(context.Context, string, ...string) error

func Reload(ctx context.Context, run CommandRunner) error {
	if err := run(ctx, "hyprctl", "reload"); err != nil {
		return fmt.Errorf("run hyprctl reload: %w", err)
	}

	return nil
}

func SystemRunner(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run()
}
