package config_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Warashi/git-wit/internal/config"
	"github.com/Warashi/git-wit/internal/git"
	"github.com/Warashi/git-wit/internal/testutil"
)

func TestLoadDefaults(t *testing.T) {
	repoDir := testutil.InitGitRepo(t)
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	cfg, err := config.Load(context.Background(), git.NewRunner(repoDir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.IgnoredDefault != config.ModeNone {
		t.Fatalf("IgnoredDefault = %q, want %q", cfg.IgnoredDefault, config.ModeNone)
	}

	if cfg.UntrackedDefault != config.ModeNone {
		t.Fatalf("UntrackedDefault = %q, want %q", cfg.UntrackedDefault, config.ModeNone)
	}

	wantRoot := filepath.Join(dataHome, "git-wit", "worktrees")
	if cfg.WorktreeRoot != wantRoot {
		t.Fatalf("WorktreeRoot = %q, want %q", cfg.WorktreeRoot, wantRoot)
	}
}

func TestLoadConfiguredValues(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	customRoot := filepath.Join(t.TempDir(), "managed-worktrees")
	testutil.RunGit(t, repoDir, "config", "wit.ignored", "symlink")
	testutil.RunGit(t, repoDir, "config", "wit.untracked", "copy")
	testutil.RunGit(t, repoDir, "config", "--add", "wit.nosync.path", "*.log")
	testutil.RunGit(t, repoDir, "config", "--add", "wit.symlink.path", "node_modules")
	testutil.RunGit(t, repoDir, "config", "--add", "wit.copy.path", ".env.local")
	testutil.RunGit(t, repoDir, "config", "wit.worktree.root", customRoot)

	cfg, err := config.Load(context.Background(), git.NewRunner(repoDir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	assertConfiguredModes(t, cfg)
	assertConfiguredPaths(t, cfg)

	if cfg.WorktreeRoot != customRoot {
		t.Fatalf("WorktreeRoot = %q, want %q", cfg.WorktreeRoot, customRoot)
	}
}

func assertConfiguredModes(t *testing.T, cfg config.Config) {
	t.Helper()

	if cfg.IgnoredDefault != config.ModeSymlink {
		t.Fatalf("IgnoredDefault = %q, want %q", cfg.IgnoredDefault, config.ModeSymlink)
	}

	if cfg.UntrackedDefault != config.ModeCopy {
		t.Fatalf("UntrackedDefault = %q, want %q", cfg.UntrackedDefault, config.ModeCopy)
	}
}

func assertConfiguredPaths(t *testing.T, cfg config.Config) {
	t.Helper()

	if len(cfg.NoSyncPaths) != 1 || cfg.NoSyncPaths[0] != "*.log" {
		t.Fatalf("NoSyncPaths = %#v", cfg.NoSyncPaths)
	}

	if len(cfg.SymlinkPaths) != 1 || cfg.SymlinkPaths[0] != "node_modules" {
		t.Fatalf("SymlinkPaths = %#v", cfg.SymlinkPaths)
	}

	if len(cfg.CopyPaths) != 1 || cfg.CopyPaths[0] != ".env.local" {
		t.Fatalf("CopyPaths = %#v", cfg.CopyPaths)
	}
}
