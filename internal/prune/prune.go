// Package prune implements git-wit prune.
package prune

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Warashi/git-wit/internal/git"
	"github.com/Warashi/git-wit/internal/metadata"
	"github.com/Warashi/git-wit/internal/repository"
	"github.com/Warashi/git-wit/internal/worktree"
)

var errWorktreeOwnerUnknown = errors.New("worktree owner unknown")

// Result summarizes prune actions and warnings.
type Result struct {
	BrokenSymlinks []string
	OrphanDirs     []string
	RemovedDirs    []string
	RemovedRefs    []string
}

// Run reconciles refs and managed worktree directories.
func Run(ctx context.Context, cwd string) (Result, error) {
	repo, err := repository.Discover(ctx, cwd)
	if err != nil {
		return Result{}, fmt.Errorf("discover repository: %w", err)
	}

	items, err := metadata.List(ctx, repo.Runner())
	if err != nil {
		return Result{}, fmt.Errorf("list metadata: %w", err)
	}

	result := Result{
		BrokenSymlinks: []string{},
		OrphanDirs:     []string{},
		RemovedDirs:    []string{},
		RemovedRefs:    []string{},
	}

	refIDs, err := pruneRefs(ctx, repo, items, &result)
	if err != nil {
		return Result{}, err
	}

	err = collectOrphanDirs(repo, refIDs, &result)
	if err != nil {
		return Result{}, err
	}

	return result, nil
}

// ScanSystem finds orphaned managed worktree directories under the configured root.
func ScanSystem(ctx context.Context, cwd string) (Result, error) {
	repo, err := repository.Discover(ctx, cwd)
	if err != nil {
		return Result{}, fmt.Errorf("discover repository: %w", err)
	}

	result := Result{
		BrokenSymlinks: []string{},
		OrphanDirs:     []string{},
		RemovedDirs:    []string{},
		RemovedRefs:    []string{},
	}

	entries, err := os.ReadDir(repo.WorktreeRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}

		return Result{}, fmt.Errorf("read worktree root: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		worktreeID := entry.Name()
		if worktree.ValidateID(worktreeID) != nil {
			continue
		}

		worktreePath := filepath.Join(repo.WorktreeRoot(), worktreeID)

		ownerErr := resolveWorktreeOwner(ctx, worktreePath)
		if errors.Is(ownerErr, errWorktreeOwnerUnknown) {
			result.OrphanDirs = append(result.OrphanDirs, worktreePath)

			continue
		}

		if ownerErr != nil {
			return Result{}, ownerErr
		}

		brokenLinks, err := findBrokenSymlinks(worktreePath)
		if err != nil {
			return Result{}, err
		}

		result.BrokenSymlinks = append(result.BrokenSymlinks, brokenLinks...)
	}

	return result, nil
}

// RemoveSystemOrphans deletes orphaned worktree directories discovered by ScanSystem.
func RemoveSystemOrphans(orphanDirs []string) ([]string, error) {
	removed := make([]string, 0, len(orphanDirs))

	for _, dirPath := range orphanDirs {
		err := os.RemoveAll(dirPath)
		if err != nil {
			return removed, fmt.Errorf("remove orphan dir: %w", err)
		}

		removed = append(removed, dirPath)
	}

	return removed, nil
}

func pruneRefs(
	ctx context.Context,
	repo repository.Repository,
	items []metadata.Metadata,
	result *Result,
) (map[string]struct{}, error) {
	refIDs := make(map[string]struct{}, len(items))

	for _, item := range items {
		refIDs[item.ID] = struct{}{}

		worktreePath := repo.WorktreePath(item.ID)

		_, statErr := os.Stat(worktreePath)
		if os.IsNotExist(statErr) {
			err := metadata.Delete(ctx, repo.Runner(), item.ID)
			if err != nil {
				return nil, fmt.Errorf("delete orphan ref: %w", err)
			}

			result.RemovedRefs = append(result.RemovedRefs, item.ID)

			continue
		}

		if statErr != nil {
			return nil, fmt.Errorf("stat managed worktree: %w", statErr)
		}

		brokenLinks, err := findBrokenSymlinks(worktreePath)
		if err != nil {
			return nil, err
		}

		result.BrokenSymlinks = append(result.BrokenSymlinks, brokenLinks...)
	}

	return refIDs, nil
}

func collectOrphanDirs(repo repository.Repository, refIDs map[string]struct{}, result *Result) error {
	entries, err := os.ReadDir(repo.WorktreeRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("read worktree root: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if _, ok := refIDs[entry.Name()]; ok {
			continue
		}

		result.OrphanDirs = append(result.OrphanDirs, filepath.Join(repo.WorktreeRoot(), entry.Name()))
	}

	return nil
}

func findBrokenSymlinks(root string) ([]string, error) {
	broken := []string{}

	err := filepath.WalkDir(root, func(currentPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type()&os.ModeSymlink == 0 {
			return nil
		}

		_, statErr := os.Stat(currentPath)
		if os.IsNotExist(statErr) {
			broken = append(broken, currentPath)

			return nil
		}

		if statErr != nil {
			return fmt.Errorf("stat symlink target: %w", statErr)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk managed worktree: %w", err)
	}

	return broken, nil
}

func resolveWorktreeOwner(ctx context.Context, worktreePath string) error {
	result, err := git.NewRunner(worktreePath).Run(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return errWorktreeOwnerUnknown
	}

	if result.Stdout == "" {
		return errWorktreeOwnerUnknown
	}

	return nil
}
