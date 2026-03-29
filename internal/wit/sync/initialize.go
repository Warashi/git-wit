package sync

import (
	"context"
	"fmt"
	"io"

	"github.com/Warashi/git-wit/internal/git"
)

// Initialize applies add-time synchronization and hooks for a new managed worktree.
func Initialize(
	ctx context.Context,
	runner git.Runner,
	repoRoot string,
	destRoot string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	cfg, err := Load(ctx, runner)
	if err != nil {
		return fmt.Errorf("load sync config: %w", err)
	}

	err = Apply(ctx, runner, repoRoot, destRoot, cfg)
	if err != nil {
		return fmt.Errorf("apply sync snapshot: %w", err)
	}

	if len(cfg.AddHooks) == 0 {
		return nil
	}

	err = Run(ctx, destRoot, cfg.AddHooks, stdout, stderr)
	if err != nil {
		return fmt.Errorf("run add hooks: %w", err)
	}

	return nil
}
