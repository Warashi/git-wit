// Package snapshot handles one-time synchronization into managed worktrees.
package snapshot

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Warashi/git-wit/internal/config"
	"github.com/Warashi/git-wit/internal/git"
)

const (
	copyParentPerm    = 0o750
	managedRootPrefix = ".git-wit/"
)

// Kind describes how Git classifies the source path.
type Kind string

const (
	// KindIgnored marks paths ignored by Git.
	KindIgnored Kind = "ignored"
	// KindUntracked marks paths not tracked by Git.
	KindUntracked Kind = "untracked"
)

// Item is a synchronization candidate.
type Item struct {
	Path string
	Kind Kind
}

// Apply copies or symlinks the synchronized snapshot into destRoot.
func Apply(ctx context.Context, runner git.Runner, repoRoot string, destRoot string, cfg config.Config) error {
	items, err := Collect(ctx, runner)
	if err != nil {
		return err
	}

	for _, item := range items {
		mode := Resolve(cfg, item.Path, item.Kind)
		if mode == config.ModeNone {
			continue
		}

		srcPath := filepath.Join(repoRoot, filepath.FromSlash(item.Path))

		destPath := filepath.Join(destRoot, filepath.FromSlash(item.Path))
		if mode == config.ModeCopy {
			err = copyPath(srcPath, destPath)
			if err != nil {
				return fmt.Errorf("copy %s: %w", item.Path, err)
			}

			continue
		}

		err = symlinkPath(srcPath, destPath)
		if err != nil {
			return fmt.Errorf("symlink %s: %w", item.Path, err)
		}
	}

	return nil
}

// Collect lists ignored and untracked synchronization candidates.
func Collect(ctx context.Context, runner git.Runner) ([]Item, error) {
	untracked, err := collectUntracked(ctx, runner)
	if err != nil {
		return nil, err
	}

	ignored, err := collectIgnored(ctx, runner)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(untracked)+len(ignored))

	items := make([]Item, 0, len(untracked)+len(ignored))
	for _, item := range append(ignored, untracked...) {
		if isManagedPath(item.Path) {
			continue
		}

		if _, ok := seen[item.Path]; ok {
			continue
		}

		seen[item.Path] = struct{}{}
		items = append(items, item)
	}

	slices.SortFunc(items, func(a Item, b Item) int {
		return strings.Compare(a.Path, b.Path)
	})

	return items, nil
}

// Resolve determines the synchronization mode for a repo-relative path.
func Resolve(cfg config.Config, itemPath string, kind Kind) config.Mode {
	if matchesAny(cfg.NoSyncPaths, itemPath) {
		return config.ModeNone
	}

	if matchesAny(cfg.SymlinkPaths, itemPath) {
		return config.ModeSymlink
	}

	if matchesAny(cfg.CopyPaths, itemPath) {
		return config.ModeCopy
	}

	if kind == KindIgnored {
		return cfg.IgnoredDefault
	}

	return cfg.UntrackedDefault
}

func collectUntracked(ctx context.Context, runner git.Runner) ([]Item, error) {
	result, err := runner.Run(ctx, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, fmt.Errorf("list untracked files: %w", err)
	}

	if result.Stdout == "" {
		return nil, nil
	}

	lines := strings.Split(result.Stdout, "\n")

	items := make([]Item, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}

		items = append(items, Item{
			Path: line,
			Kind: KindUntracked,
		})
	}

	return items, nil
}

func collectIgnored(ctx context.Context, runner git.Runner) ([]Item, error) {
	result, err := runner.Run(ctx, "status", "--porcelain=v1", "--ignored=matching")
	if err != nil {
		return nil, fmt.Errorf("list ignored files: %w", err)
	}

	if result.Stdout == "" {
		return nil, nil
	}

	lines := strings.Split(result.Stdout, "\n")

	items := make([]Item, 0, len(lines))
	for _, line := range lines {
		if !strings.HasPrefix(line, "!! ") {
			continue
		}

		items = append(items, Item{
			Path: strings.TrimPrefix(line, "!! "),
			Kind: KindIgnored,
		})
	}

	return items, nil
}

func matchesAny(patterns []string, itemPath string) bool {
	for _, pattern := range patterns {
		if matchesPattern(pattern, itemPath) {
			return true
		}
	}

	return false
}

func matchesPattern(pattern string, itemPath string) bool {
	matched, err := path.Match(pattern, itemPath)
	if err == nil && matched {
		return true
	}

	if !strings.ContainsAny(pattern, "*?[") {
		return itemPath == pattern || strings.HasPrefix(itemPath, pattern+"/")
	}

	return false
}

func isManagedPath(itemPath string) bool {
	return itemPath == ".git-wit" || strings.HasPrefix(itemPath, managedRootPrefix)
}

func copyPath(srcPath string, destPath string) error {
	info, err := os.Lstat(srcPath)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return copySymlink(srcPath, destPath)
	}

	if info.IsDir() {
		return copyDir(srcPath, destPath, info.Mode())
	}

	return copyFile(srcPath, destPath, info.Mode())
}

func copyDir(srcPath string, destPath string, mode fs.FileMode) error {
	err := os.MkdirAll(destPath, mode.Perm())
	if err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}

	for _, entry := range entries {
		srcChild := filepath.Join(srcPath, entry.Name())
		destChild := filepath.Join(destPath, entry.Name())

		err = copyPath(srcChild, destChild)
		if err != nil {
			return err
		}
	}

	return nil
}

func copyFile(srcPath string, destPath string, mode fs.FileMode) error {
	err := os.MkdirAll(filepath.Dir(destPath), copyParentPerm)
	if err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	// #nosec G304 -- source paths are discovered from the active repository.
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}

	defer func() {
		_ = srcFile.Close()
	}()

	// #nosec G304 -- destination paths are under the managed worktree root.
	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return fmt.Errorf("open destination file: %w", err)
	}

	_, err = io.Copy(destFile, srcFile)
	closeErr := destFile.Close()

	if err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	if closeErr != nil {
		return fmt.Errorf("close destination file: %w", closeErr)
	}

	return nil
}

func copySymlink(srcPath string, destPath string) error {
	target, err := os.Readlink(srcPath)
	if err != nil {
		return fmt.Errorf("read source symlink: %w", err)
	}

	err = os.MkdirAll(filepath.Dir(destPath), copyParentPerm)
	if err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	err = os.Symlink(target, destPath)
	if err != nil {
		return fmt.Errorf("create destination symlink: %w", err)
	}

	return nil
}

func symlinkPath(srcPath string, destPath string) error {
	err := os.MkdirAll(filepath.Dir(destPath), copyParentPerm)
	if err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	relativeTarget, err := filepath.Rel(filepath.Dir(destPath), srcPath)
	if err != nil {
		return fmt.Errorf("compute relative symlink target: %w", err)
	}

	err = os.Symlink(relativeTarget, destPath)
	if err != nil {
		return fmt.Errorf("create symlink: %w", err)
	}

	return nil
}
