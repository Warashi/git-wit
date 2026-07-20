package reconcile

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Warashi/git-wit/internal/git"
	"github.com/Warashi/git-wit/internal/wit/catalog"
)

var errWorktreeOwnerUnknown = errors.New("worktree owner unknown")

// Result summarizes prune actions and warnings.
type Result struct {
	BrokenSymlinks []string
	OrphanDirs     []string
	RemovedDirs    []string
	RemovedRefs    []string
}

// Prune reconciles refs and managed worktree directories.
func Prune(ctx context.Context, cwd string) (Result, error) {
	repo, err := catalog.Open(ctx, cwd)
	if err != nil {
		return Result{}, fmt.Errorf("discover repository: %w", err)
	}

	items, err := repo.List(ctx)
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

	err = collectOrphanDirs(ctx, repo, refIDs, &result)
	if err != nil {
		return Result{}, err
	}

	return result, nil
}

// ScanSystem finds orphaned managed worktree directories under the configured root.
func ScanSystem(ctx context.Context, cwd string) (Result, error) {
	repo, err := catalog.Open(ctx, cwd)
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
		if catalog.ValidateID(worktreeID) != nil {
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
	repo catalog.Repository,
	items []catalog.Record,
	result *Result,
) (map[string]struct{}, error) {
	refIDs := make(map[string]struct{}, len(items))

	for _, item := range items {
		refIDs[item.ID] = struct{}{}

		worktreePath := repo.WorktreePath(item.ID)

		_, statErr := os.Stat(worktreePath)
		if os.IsNotExist(statErr) {
			err := repo.Delete(ctx, item.ID)
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

func collectOrphanDirs(ctx context.Context, repo catalog.Repository, refIDs map[string]struct{}, result *Result) error {
	entries, err := os.ReadDir(repo.WorktreeRoot())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("read worktree root: %w", err)
	}

	expected, err := git.NewRunner(repo.Root()).Run(ctx, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return fmt.Errorf("resolve common git directory: %w", err)
	}

	expectedDir := strings.TrimSpace(expected.Stdout)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if _, ok := refIDs[entry.Name()]; ok {
			continue
		}

		worktreePath := filepath.Join(repo.WorktreeRoot(), entry.Name())

		if isWorktreeOf(ctx, worktreePath, expectedDir) {
			result.OrphanDirs = append(result.OrphanDirs, worktreePath)
		}
	}

	return nil
}

func isWorktreeOf(ctx context.Context, worktreePath string, expectedGitCommonDir string) bool {
	result, err := git.NewRunner(worktreePath).Run(ctx, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return false
	}

	return strings.TrimSpace(result.Stdout) == expectedGitCommonDir
}

func findBrokenSymlinks(root string) ([]string, error) {
	broken := []string{}

	err := filepath.WalkDir(root, func(currentPath string, entry fs.DirEntry, err error) error {
		// Unreadable entries (e.g. permission-restricted subdirectories)
		// must not abort the whole scan: the symlink report is advisory
		// and prune has other work to finish.
		if err != nil {
			return nil //nolint:nilerr // Skipping unreadable entries is the tolerant behavior wanted here.
		}

		if entry.Type()&os.ModeSymlink == 0 {
			return nil
		}

		_, statErr := os.Stat(currentPath)
		if os.IsNotExist(statErr) {
			broken = append(broken, currentPath)
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
		// A cancelled context fails every probe the same way; classifying
		// that as "orphan" would queue perfectly healthy worktrees for
		// deletion.
		if ctx.Err() != nil {
			return fmt.Errorf("resolve worktree owner: %w", ctx.Err())
		}

		return errWorktreeOwnerUnknown
	}

	if result.Stdout == "" {
		return errWorktreeOwnerUnknown
	}

	return nil
}
