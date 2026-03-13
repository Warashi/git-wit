// Package repository resolves the active git repository context.
package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Warashi/git-wit/internal/config"
	"github.com/Warashi/git-wit/internal/git"
	"github.com/Warashi/git-wit/internal/worktree"
)

const worktreeRootPerm = 0o750

// Repository describes the resolved git-wit repository context.
type Repository struct {
	root         string
	runner       git.Runner
	worktreeRoot string
}

// Discover resolves the current repository from cwd.
func Discover(ctx context.Context, cwd string) (Repository, error) {
	runner := git.NewRunner(cwd)

	result, err := runner.Run(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return Repository{}, fmt.Errorf("discover repository root: %w", err)
	}

	root, err := filepath.Abs(result.Stdout)
	if err != nil {
		return Repository{}, fmt.Errorf("normalize repository root: %w", err)
	}

	repoRunner := git.NewRunner(root)

	cfg, err := config.Load(ctx, repoRunner)
	if err != nil {
		return Repository{}, fmt.Errorf("load config: %w", err)
	}

	return Repository{
		root:         root,
		runner:       repoRunner,
		worktreeRoot: cfg.WorktreeRoot,
	}, nil
}

// EnsureWorktreeRoot creates the managed worktree root if needed.
func (r Repository) EnsureWorktreeRoot() error {
	err := os.MkdirAll(r.WorktreeRoot(), worktreeRootPerm)
	if err != nil {
		return fmt.Errorf("create worktree root: %w", err)
	}

	return nil
}

// Root returns the absolute repository root.
func (r Repository) Root() string {
	return r.root
}

// Runner returns a git runner rooted at the repository.
func (r Repository) Runner() git.Runner {
	return r.runner
}

// WorktreeRoot returns the managed worktree directory.
func (r Repository) WorktreeRoot() string {
	return r.worktreeRoot
}

// WorktreePath returns the path for a managed worktree ID.
func (r Repository) WorktreePath(id string) string {
	return worktree.Path(r.worktreeRoot, id)
}
