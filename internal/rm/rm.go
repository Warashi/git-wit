// Package rm implements git-wit rm.
package rm

import (
	"context"
	"errors"
	"fmt"

	"github.com/Warashi/git-wit/internal/metadata"
	"github.com/Warashi/git-wit/internal/repository"
	"github.com/Warashi/git-wit/internal/worktree"
)

var errUnknownWorktreeID = errors.New("unknown worktree id")

// Run removes a managed worktree and its metadata ref.
func Run(ctx context.Context, cwd string, worktreeID string) error {
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

	_, err = repo.Runner().Run(ctx, "worktree", "remove", repo.WorktreePath(worktreeID))
	if err != nil {
		return fmt.Errorf("remove worktree: %w", err)
	}

	err = metadata.Delete(ctx, repo.Runner(), worktreeID)
	if err != nil {
		return fmt.Errorf("delete metadata ref: %w", err)
	}

	return nil
}
