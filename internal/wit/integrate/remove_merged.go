package integrate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Warashi/git-wit/internal/wit/catalog"
	"github.com/Warashi/git-wit/internal/wit/query"
)

var errWorktreeNoLongerIntegrated = errors.New("worktree is no longer integrated")

// Candidate describes an integrated managed worktree eligible for removal.
type Candidate struct {
	ID   string
	Path string
}

// RemovalResult reports the outcome of one bulk removal attempt.
type RemovalResult struct {
	Candidate Candidate
	Err       error
}

// MergedCandidates returns safely integrated worktrees, excluding the active
// managed worktree.
func MergedCandidates(ctx context.Context, cwd string) ([]Candidate, error) {
	repo, err := catalog.Open(ctx, cwd)
	if err != nil {
		return nil, fmt.Errorf("discover repository: %w", err)
	}

	entries, err := query.List(ctx, cwd)
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}

	candidates := make([]Candidate, 0, len(entries))
	for _, entry := range entries {
		if !entry.Integrated || samePath(entry.Path, repo.Root()) {
			continue
		}

		candidates = append(candidates, Candidate{
			ID:   entry.ID,
			Path: entry.Path,
		})
	}

	return candidates, nil
}

// RemoveMerged removes candidates that are still safely integrated, continuing
// after individual failures.
func RemoveMerged(
	ctx context.Context,
	cwd string,
	candidates []Candidate,
	stderr io.Writer,
) []RemovalResult {
	results := make([]RemovalResult, 0, len(candidates))

	for _, candidate := range candidates {
		err := validateMergedCandidate(ctx, cwd, candidate)
		if err == nil {
			err = Remove(ctx, cwd, candidate.ID, nil, stderr)
		}

		results = append(results, RemovalResult{
			Candidate: candidate,
			Err:       err,
		})
	}

	return results
}

func validateMergedCandidate(ctx context.Context, cwd string, candidate Candidate) error {
	entry, err := query.Get(ctx, cwd, candidate.ID)
	if err != nil {
		return fmt.Errorf("refresh worktree state: %w", err)
	}

	if entry.Path != candidate.Path || !entry.Integrated {
		return errWorktreeNoLongerIntegrated
	}

	repo, err := catalog.Open(ctx, cwd)
	if err != nil {
		return fmt.Errorf("discover repository: %w", err)
	}

	if samePath(entry.Path, repo.Root()) {
		return errWorktreeNoLongerIntegrated
	}

	return nil
}

func samePath(left string, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}
