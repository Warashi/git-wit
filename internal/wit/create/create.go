package create

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/Warashi/git-wit/internal/wit/catalog"
	"github.com/Warashi/git-wit/internal/wit/sync"
)

// Result describes a created managed worktree.
type Result struct {
	ID   string
	Path string
}

// Create creates a managed worktree and synchronizes the initial snapshot.
func Create(
	ctx context.Context,
	cwd string,
	now time.Time,
	memo string,
	_ io.Writer,
	stderr io.Writer,
) (Result, error) {
	repo, err := catalog.Open(ctx, cwd)
	if err != nil {
		return Result{}, fmt.Errorf("discover repository: %w", err)
	}

	baseResult, err := repo.Runner().Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return Result{}, fmt.Errorf("resolve base commit: %w", err)
	}

	record := catalog.NewRecord(now, memo, baseResult.Stdout)

	err = repo.Store(ctx, record)
	if err != nil {
		return Result{}, fmt.Errorf("store metadata: %w", err)
	}

	err = repo.EnsureWorktreeRoot()
	if err != nil {
		return Result{}, fmt.Errorf("ensure worktree root: %w", err)
	}

	worktreePath := repo.WorktreePath(record.ID)

	_, err = repo.Runner().WithStreams(stderr, stderr).Run(ctx, "worktree", "add", "-d", worktreePath)
	if err != nil {
		return Result{}, fmt.Errorf("create worktree: %w", err)
	}

	err = sync.Initialize(ctx, repo.Runner(), repo.Root(), worktreePath, stderr, stderr)
	if err != nil {
		return Result{}, fmt.Errorf("initialize worktree sync: %w", err)
	}

	return Result{
		ID:   record.ID,
		Path: worktreePath,
	}, nil
}
