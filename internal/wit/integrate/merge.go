package integrate

import (
	"context"
	"fmt"
	"io"

	"github.com/Warashi/git-wit/internal/wit/catalog"
)

// Merge merges the worktree HEAD into the current branch.
func Merge(ctx context.Context, cwd string, worktreeID string, remove bool, stdout io.Writer, stderr io.Writer) error {
	err := catalog.ValidateID(worktreeID)
	if err != nil {
		return fmt.Errorf("validate id: %w", err)
	}

	repo, err := catalog.Open(ctx, cwd)
	if err != nil {
		return fmt.Errorf("discover repository: %w", err)
	}

	exists, err := repo.Exists(ctx, worktreeID)
	if err != nil {
		return fmt.Errorf("check metadata ref: %w", err)
	}

	if !exists {
		return fmt.Errorf("%w: %s", errUnknownWorktreeID, worktreeID)
	}

	targetRunner := repo.Runner()
	targetRunner = targetRunner.WithRepoDir(repo.WorktreePath(worktreeID))

	result, err := targetRunner.Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("resolve worktree head: %w", err)
	}

	_, err = repo.Runner().WithStreams(stdout, stderr).Run(ctx, "merge", result.Stdout)
	if err != nil {
		return fmt.Errorf("merge worktree head: %w", err)
	}

	if remove {
		err = Remove(ctx, cwd, worktreeID, false, stdout, stderr)
		if err != nil {
			return fmt.Errorf("remove merged worktree: %w", err)
		}
	}

	return nil
}
