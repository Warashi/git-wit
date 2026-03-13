package config_test

import (
	"context"
	"testing"

	"github.com/Warashi/git-wit/internal/config"
	"github.com/Warashi/git-wit/internal/git"
	"github.com/Warashi/git-wit/internal/testutil"
)

func TestLoadDefaults(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)

	cfg, err := config.Load(context.Background(), git.NewRunner(repoDir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.IgnoredDefault != config.ModeCopy {
		t.Fatalf("IgnoredDefault = %q, want %q", cfg.IgnoredDefault, config.ModeCopy)
	}

	if cfg.UntrackedDefault != config.ModeNone {
		t.Fatalf("UntrackedDefault = %q, want %q", cfg.UntrackedDefault, config.ModeNone)
	}
}

func TestLoadConfiguredValues(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	testutil.RunGit(t, repoDir, "config", "wit.ignored", "symlink")
	testutil.RunGit(t, repoDir, "config", "wit.untracked", "copy")
	testutil.RunGit(t, repoDir, "config", "--add", "wit.nosync.path", "*.log")
	testutil.RunGit(t, repoDir, "config", "--add", "wit.symlink.path", "node_modules")
	testutil.RunGit(t, repoDir, "config", "--add", "wit.copy.path", ".env.local")

	cfg, err := config.Load(context.Background(), git.NewRunner(repoDir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.IgnoredDefault != config.ModeSymlink {
		t.Fatalf("IgnoredDefault = %q, want %q", cfg.IgnoredDefault, config.ModeSymlink)
	}

	if cfg.UntrackedDefault != config.ModeCopy {
		t.Fatalf("UntrackedDefault = %q, want %q", cfg.UntrackedDefault, config.ModeCopy)
	}

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
