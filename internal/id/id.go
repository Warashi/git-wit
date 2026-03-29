// Package id implements git-wit id.
package id

import (
	"context"
	"errors"
	"fmt"

	"github.com/Warashi/git-wit/internal/wit/catalog"
)

var errNotManagedWorktree = errors.New("not in a managed worktree")

// Run returns the current managed worktree ID.
func Run(ctx context.Context, cwd string) (string, error) {
	repo, err := catalog.Open(ctx, cwd)
	if err != nil {
		return "", fmt.Errorf("discover repository: %w", err)
	}

	worktreeID, err := repo.CurrentID()
	if err != nil {
		return "", fmt.Errorf("resolve current worktree id: %w", err)
	}

	exists, err := repo.Exists(ctx, worktreeID)
	if err != nil {
		return "", fmt.Errorf("check metadata ref: %w", err)
	}

	if !exists {
		return "", fmt.Errorf("%w: %s", errNotManagedWorktree, worktreeID)
	}

	return worktreeID, nil
}
