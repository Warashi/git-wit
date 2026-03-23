// Package id implements git-wit id.
package id

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Warashi/git-wit/internal/metadata"
	"github.com/Warashi/git-wit/internal/repository"
	"github.com/Warashi/git-wit/internal/worktree"
)

var errNotManagedWorktree = errors.New("not in a managed worktree")

// Run returns the current managed worktree ID.
func Run(ctx context.Context, cwd string) (string, error) {
	repo, err := repository.Discover(ctx, cwd)
	if err != nil {
		return "", fmt.Errorf("discover repository: %w", err)
	}

	worktreeID, err := resolveWorktreeID(repo)
	if err != nil {
		return "", err
	}

	exists, err := metadata.Exists(ctx, repo.Runner(), worktreeID)
	if err != nil {
		return "", fmt.Errorf("check metadata ref: %w", err)
	}

	if !exists {
		return "", fmt.Errorf("%w: %s", errNotManagedWorktree, worktreeID)
	}

	return worktreeID, nil
}

func resolveWorktreeID(repo repository.Repository) (string, error) {
	rel, err := filepath.Rel(repo.WorktreeRoot(), repo.Root())
	if err != nil {
		return "", fmt.Errorf("rel worktree path: %w", err)
	}

	if rel == "." || filepath.Dir(rel) != "." {
		return "", errNotManagedWorktree
	}

	validationErr := worktree.ValidateID(rel)
	if validationErr != nil {
		return "", fmt.Errorf("validate worktree id: %w", validationErr)
	}

	return rel, nil
}
