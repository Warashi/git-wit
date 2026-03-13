// Package dir implements git-wit dir.
package dir

import (
	"context"
	"errors"
	"fmt"

	"github.com/Warashi/git-wit/internal/metadata"
	"github.com/Warashi/git-wit/internal/repository"
	"github.com/Warashi/git-wit/internal/worktree"
)

var errUnknownWorktreeID = errors.New("unknown worktree id")

// Run prints the managed worktree path for an ID.
func Run(ctx context.Context, cwd string, worktreeID string) (string, error) {
	err := worktree.ValidateID(worktreeID)
	if err != nil {
		return "", fmt.Errorf("validate id: %w", err)
	}

	repo, err := repository.Discover(ctx, cwd)
	if err != nil {
		return "", fmt.Errorf("discover repository: %w", err)
	}

	exists, err := metadata.Exists(ctx, repo.Runner(), worktreeID)
	if err != nil {
		return "", fmt.Errorf("check metadata ref: %w", err)
	}

	if !exists {
		return "", fmt.Errorf("%w: %s", errUnknownWorktreeID, worktreeID)
	}

	return repo.WorktreePath(worktreeID), nil
}
