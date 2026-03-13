// Package ls implements git-wit ls.
package ls

import (
	"context"
	"fmt"
	"time"

	"github.com/Warashi/git-wit/internal/metadata"
	"github.com/Warashi/git-wit/internal/repository"
)

// Entry is one row of ls output.
type Entry struct {
	ID        string
	CreatedAt time.Time
	Path      string
	Memo      string
}

// Run lists managed worktrees.
func Run(ctx context.Context, cwd string) ([]Entry, error) {
	repo, err := repository.Discover(ctx, cwd)
	if err != nil {
		return nil, fmt.Errorf("discover repository: %w", err)
	}

	items, err := metadata.List(ctx, repo.Runner())
	if err != nil {
		return nil, fmt.Errorf("list metadata: %w", err)
	}

	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		entries = append(entries, Entry{
			ID:        item.ID,
			CreatedAt: item.CreatedAt,
			Path:      repo.WorktreePath(item.ID),
			Memo:      item.Memo,
		})
	}

	return entries, nil
}
