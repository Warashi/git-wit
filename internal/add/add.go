// Package add implements git-wit add.
package add

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Warashi/git-wit/internal/config"
	"github.com/Warashi/git-wit/internal/hook"
	"github.com/Warashi/git-wit/internal/metadata"
	"github.com/Warashi/git-wit/internal/repository"
	"github.com/Warashi/git-wit/internal/snapshot"
	"github.com/Warashi/git-wit/internal/worktree"
)

// Result describes a created managed worktree.
type Result struct {
	ID   string
	Path string
}

// Run creates a managed worktree and synchronizes the initial snapshot.
func Run(ctx context.Context, cwd string, now time.Time, memo string) (Result, error) {
	repo, err := repository.Discover(ctx, cwd)
	if err != nil {
		return Result{}, fmt.Errorf("discover repository: %w", err)
	}

	worktreeID := worktree.NewID()
	item := metadata.New(worktreeID, now, memo)

	err = metadata.Store(ctx, repo.Runner(), item)
	if err != nil {
		return Result{}, fmt.Errorf("store metadata: %w", err)
	}

	err = repo.EnsureWorktreeRoot()
	if err != nil {
		return Result{}, fmt.Errorf("ensure worktree root: %w", err)
	}

	worktreePath := repo.WorktreePath(worktreeID)

	_, err = repo.Runner().Run(ctx, "worktree", "add", "-d", worktreePath)
	if err != nil {
		return Result{}, fmt.Errorf("create worktree: %w", err)
	}

	cfg, err := config.Load(ctx, repo.Runner())
	if err != nil {
		return Result{}, fmt.Errorf("load snapshot config: %w", err)
	}

	err = snapshot.Apply(ctx, repo.Runner(), repo.Root(), worktreePath, cfg)
	if err != nil {
		return Result{}, fmt.Errorf("apply snapshot: %w", err)
	}

	if len(cfg.AddHooks) > 0 {
		err = hook.Run(ctx, worktreePath, cfg.AddHooks, os.Stdout, os.Stderr)
		if err != nil {
			return Result{}, fmt.Errorf("run add hooks: %w", err)
		}
	}

	return Result{
		ID:   worktreeID,
		Path: worktreePath,
	}, nil
}
