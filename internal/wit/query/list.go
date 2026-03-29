package query

import (
	"context"
	"fmt"
	"time"

	"github.com/Warashi/git-wit/internal/wit/catalog"
)

// Entry is one row of ls output.
type Entry struct {
	ID        string
	CreatedAt time.Time
	Path      string
	Memo      string
}

// List lists managed worktrees.
func List(ctx context.Context, cwd string) ([]Entry, error) {
	repo, err := catalog.Open(ctx, cwd)
	if err != nil {
		return nil, fmt.Errorf("discover repository: %w", err)
	}

	items, err := repo.List(ctx)
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
