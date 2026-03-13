// Package merge implements git-wit merge.
package merge

import (
	"context"
	"errors"
	"fmt"

	"github.com/Warashi/git-wit/internal/metadata"
	"github.com/Warashi/git-wit/internal/repository"
	removewt "github.com/Warashi/git-wit/internal/rm"
	"github.com/Warashi/git-wit/internal/worktree"
)

var errUnknownWorktreeID = errors.New("unknown worktree id")

// Run merges the worktree HEAD into the current branch.
func Run(ctx context.Context, cwd string, worktreeID string, remove bool) error {
	err := worktree.ValidateID(worktreeID)
	if err != nil {
		return fmt.Errorf("validate id: %w", err)
	}

	repo, err := repository.Discover(ctx, cwd)
	if err != nil {
		return fmt.Errorf("discover repository: %w", err)
	}

	exists, err := metadata.Exists(ctx, repo.Runner(), worktreeID)
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

	_, err = repo.Runner().Run(ctx, "merge", result.Stdout)
	if err != nil {
		return fmt.Errorf("merge worktree head: %w", err)
	}

	if remove {
		err = removewt.Run(ctx, cwd, worktreeID)
		if err != nil {
			return fmt.Errorf("remove merged worktree: %w", err)
		}
	}

	return nil
}
