package query

import (
	"context"
	"fmt"

	"github.com/Warashi/git-wit/internal/wit/catalog"
)

// Memo returns the memo recorded for a managed worktree. When worktreeID is
// empty, it resolves the current managed worktree from cwd instead.
func Memo(ctx context.Context, cwd string, worktreeID string) (string, error) {
	repo, err := catalog.Open(ctx, cwd)
	if err != nil {
		return "", fmt.Errorf("discover repository: %w", err)
	}

	resolvedID := worktreeID
	notFoundErr := errUnknownWorktreeID

	if resolvedID == "" {
		resolvedID, err = repo.CurrentID()
		if err != nil {
			return "", fmt.Errorf("resolve current worktree id: %w", err)
		}

		// A resolved-but-orphaned worktree (dir exists, ref was deleted) is
		// "not managed" from the CurrentID caller's perspective, matching
		// CurrentID's own not-found semantics rather than an explicit,
		// user-supplied unknown ID.
		notFoundErr = errNotManagedWorktree
	} else {
		err = catalog.ValidateID(resolvedID)
		if err != nil {
			return "", fmt.Errorf("validate id: %w", err)
		}
	}

	exists, err := repo.Exists(ctx, resolvedID)
	if err != nil {
		return "", fmt.Errorf("check metadata ref: %w", err)
	}

	if !exists {
		return "", fmt.Errorf("%w: %s", notFoundErr, resolvedID)
	}

	record, err := repo.Load(ctx, resolvedID)
	if err != nil {
		return "", fmt.Errorf("load metadata: %w", err)
	}

	return record.Memo, nil
}
