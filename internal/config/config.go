// Package config loads git-wit settings from git config.
package config

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Warashi/git-wit/internal/git"
)

// Mode defines how a path is synchronized.
type Mode string

const (
	// ModeCopy copies files into the managed worktree.
	ModeCopy Mode = "copy"
	// ModeNone skips synchronization.
	ModeNone Mode = "none"
	// ModeSymlink creates symlinks into the source repository.
	ModeSymlink Mode = "symlink"
)

var errInvalidMode = errors.New("invalid mode")

// Config holds synchronization defaults and path overrides.
type Config struct {
	IgnoredDefault   Mode
	UntrackedDefault Mode
	NoSyncPaths      []string
	SymlinkPaths     []string
	CopyPaths        []string
}

// Load reads git-wit settings from git config.
func Load(ctx context.Context, runner git.Runner) (Config, error) {
	ignoredDefault, err := loadMode(ctx, runner, "wit.ignored", ModeCopy)
	if err != nil {
		return Config{}, err
	}

	untrackedDefault, err := loadMode(ctx, runner, "wit.untracked", ModeNone)
	if err != nil {
		return Config{}, err
	}

	noSyncPaths, err := loadMultiValue(ctx, runner, "wit.nosync.path")
	if err != nil {
		return Config{}, err
	}

	symlinkPaths, err := loadMultiValue(ctx, runner, "wit.symlink.path")
	if err != nil {
		return Config{}, err
	}

	copyPaths, err := loadMultiValue(ctx, runner, "wit.copy.path")
	if err != nil {
		return Config{}, err
	}

	return Config{
		IgnoredDefault:   ignoredDefault,
		UntrackedDefault: untrackedDefault,
		NoSyncPaths:      noSyncPaths,
		SymlinkPaths:     symlinkPaths,
		CopyPaths:        copyPaths,
	}, nil
}

func loadMode(ctx context.Context, runner git.Runner, key string, fallback Mode) (Mode, error) {
	result, err := runner.Run(ctx, "config", "--get", key)
	if err != nil {
		var cmdErr git.CommandError
		if isMissingConfig(err, &cmdErr) {
			return fallback, nil
		}

		return "", fmt.Errorf("load %s: %w", key, err)
	}

	mode := Mode(strings.TrimSpace(result.Stdout))
	switch mode {
	case ModeCopy, ModeNone, ModeSymlink:
		return mode, nil
	default:
		return "", fmt.Errorf("load %s: %w %q", key, errInvalidMode, result.Stdout)
	}
}

func loadMultiValue(ctx context.Context, runner git.Runner, key string) ([]string, error) {
	result, err := runner.Run(ctx, "config", "--get-all", key)
	if err != nil {
		var cmdErr git.CommandError
		if isMissingConfig(err, &cmdErr) {
			return nil, nil
		}

		return nil, fmt.Errorf("load %s: %w", key, err)
	}

	if result.Stdout == "" {
		return nil, nil
	}

	return strings.Split(result.Stdout, "\n"), nil
}

func isMissingConfig(err error, target *git.CommandError) bool {
	if !strings.Contains(err.Error(), " config ") {
		return false
	}

	if !errors.As(err, target) {
		return false
	}

	return target.ExitCode() == 1
}
