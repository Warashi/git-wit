package sync

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

	"github.com/Warashi/git-wit/internal/git"
)

const (
	copyParentPerm    = 0o750
	managedRootPrefix = ".wit/"
)

type dirCloneFunc func(srcPath string, destPath string) (bool, error)

type fileCloneFunc func(srcPath string, destPath string, mode fs.FileMode) (bool, error)

type copier struct {
	cloneDir  dirCloneFunc
	cloneFile fileCloneFunc
}

func newCopier() copier {
	return copier{
		cloneDir:  tryCloneDir,
		cloneFile: tryCloneFile,
	}
}

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
func Apply(ctx context.Context, runner git.Runner, repoRoot string, destRoot string, cfg Config) error {
	items, err := Collect(ctx, runner)
	if err != nil {
		return err
	}

	copier := newCopier()

	for _, item := range items {
		mode := Resolve(cfg, item.Path, item.Kind)
		if mode == ModeNone {
			continue
		}

		srcPath := filepath.Join(repoRoot, filepath.FromSlash(item.Path))

		destPath := filepath.Join(destRoot, filepath.FromSlash(item.Path))
		if mode == ModeCopy {
			err = copier.copyPath(srcPath, destPath)
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
func Resolve(cfg Config, itemPath string, kind Kind) Mode {
	if matchesAny(cfg.NoSyncPaths, itemPath) {
		return ModeNone
	}

	if matchesAny(cfg.SymlinkPaths, itemPath) {
		return ModeSymlink
	}

	if matchesAny(cfg.CopyPaths, itemPath) {
		return ModeCopy
	}

	if kind == KindIgnored {
		return cfg.IgnoredDefault
	}

	return cfg.UntrackedDefault
}

// collectUntracked and collectIgnored use -z (NUL-separated) output: without
// it git C-quotes paths containing non-ASCII bytes, quotes, or tabs (under
// the default core.quotePath), and the quoted string would be treated as a
// literal path.
func collectUntracked(ctx context.Context, runner git.Runner) ([]Item, error) {
	result, err := runner.Run(ctx, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, fmt.Errorf("list untracked files: %w", err)
	}

	entries := splitNulSeparated(result.Stdout)

	items := make([]Item, 0, len(entries))
	for _, entry := range entries {
		items = append(items, Item{
			Path: entry,
			Kind: KindUntracked,
		})
	}

	return items, nil
}

func collectIgnored(ctx context.Context, runner git.Runner) ([]Item, error) {
	result, err := runner.Run(ctx, "status", "--porcelain=v1", "-z", "--ignored=matching")
	if err != nil {
		return nil, fmt.Errorf("list ignored files: %w", err)
	}

	entries := splitNulSeparated(result.Stdout)

	items := make([]Item, 0, len(entries))
	for _, entry := range entries {
		if !strings.HasPrefix(entry, "!! ") {
			continue
		}

		items = append(items, Item{
			Path: strings.TrimPrefix(entry, "!! "),
			Kind: KindIgnored,
		})
	}

	return items, nil
}

func splitNulSeparated(raw string) []string {
	parts := strings.Split(raw, "\x00")

	entries := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}

		entries = append(entries, part)
	}

	return entries
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
	pattern = normalizeMatchPath(pattern)
	itemPath = normalizeMatchPath(itemPath)

	if !strings.ContainsAny(pattern, "*?[") {
		return itemPath == pattern || strings.HasPrefix(itemPath, pattern+"/")
	}

	return matchGlobPattern(pattern, itemPath)
}

func normalizeMatchPath(value string) string {
	return strings.TrimRight(value, "/")
}

func matchGlobPattern(pattern string, itemPath string) bool {
	patternParts := splitMatchPath(pattern)
	itemParts := splitMatchPath(itemPath)

	return matchGlobParts(patternParts, itemParts)
}

func splitMatchPath(value string) []string {
	if value == "" {
		return nil
	}

	return strings.Split(value, "/")
}

func matchGlobParts(patternParts []string, itemParts []string) bool {
	if len(patternParts) == 0 {
		return len(itemParts) == 0
	}

	if patternParts[0] == "**" {
		if matchGlobParts(patternParts[1:], itemParts) {
			return true
		}

		if len(itemParts) == 0 {
			return false
		}

		return matchGlobParts(patternParts, itemParts[1:])
	}

	if len(itemParts) == 0 {
		return false
	}

	matched, err := path.Match(patternParts[0], itemParts[0])
	if err != nil || !matched {
		return false
	}

	return matchGlobParts(patternParts[1:], itemParts[1:])
}

func isManagedPath(itemPath string) bool {
	return itemPath == ".wit" || strings.HasPrefix(itemPath, managedRootPrefix)
}

func (c copier) copyPath(srcPath string, destPath string) error {
	info, err := os.Lstat(srcPath)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return copySymlink(srcPath, destPath)
	}

	if info.IsDir() {
		return c.copyDir(srcPath, destPath, info.Mode())
	}

	// FIFOs, sockets, and devices cannot be copied as content; opening a
	// FIFO for reading would even block until a writer appears, hanging
	// the whole add.
	if !info.Mode().IsRegular() {
		return nil
	}

	return c.copyFile(srcPath, destPath, info.Mode())
}

func (c copier) copyDir(srcPath string, destPath string, mode fs.FileMode) error {
	err := os.MkdirAll(filepath.Dir(destPath), copyParentPerm)
	if err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	handled, err := c.cloneDir(srcPath, destPath)
	if err != nil {
		return fmt.Errorf("clone directory: %w", err)
	}

	if handled {
		return nil
	}

	err = os.MkdirAll(destPath, mode.Perm())
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

		err = c.copyPath(srcChild, destChild)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c copier) copyFile(srcPath string, destPath string, mode fs.FileMode) error {
	err := os.MkdirAll(filepath.Dir(destPath), copyParentPerm)
	if err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	handled, err := c.cloneFile(srcPath, destPath, mode)
	if err != nil {
		return fmt.Errorf("clone file: %w", err)
	}

	if handled {
		return nil
	}

	return copyFileContents(srcPath, destPath, mode)
}

func copyFileContents(srcPath string, destPath string, mode fs.FileMode) error {
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

	err = os.Symlink(srcPath, destPath)
	if err != nil {
		return fmt.Errorf("create symlink: %w", err)
	}

	return nil
}
