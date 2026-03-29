package integrate

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Warashi/git-wit/internal/wit/catalog"
)

var errUnknownWorktreeID = errors.New("unknown worktree id")

// Remove removes a managed worktree and its metadata ref.
func Remove(ctx context.Context, cwd string, worktreeID string, _ io.Writer, stderr io.Writer) error {
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

	_, err = repo.Runner().WithStreams(nil, stderr).Run(ctx, "worktree", "remove", repo.WorktreePath(worktreeID))
	if err != nil {
		return fmt.Errorf("remove worktree: %w", err)
	}

	err = repo.Delete(ctx, worktreeID)
	if err != nil {
		return fmt.Errorf("delete metadata ref: %w", err)
	}

	return nil
}
