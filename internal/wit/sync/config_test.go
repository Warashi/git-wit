package sync_test

import (
	"context"
	"testing"

	"github.com/Warashi/git-wit/internal/git"
	"github.com/Warashi/git-wit/internal/testutil"
	"github.com/Warashi/git-wit/internal/wit/sync"
)

func TestLoadDefaults(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)

	cfg, err := sync.Load(context.Background(), git.NewRunner(repoDir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.IgnoredDefault != sync.ModeNone {
		t.Fatalf("IgnoredDefault = %q, want %q", cfg.IgnoredDefault, sync.ModeNone)
	}

	if cfg.UntrackedDefault != sync.ModeNone {
		t.Fatalf("UntrackedDefault = %q, want %q", cfg.UntrackedDefault, sync.ModeNone)
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
	testutil.RunGit(t, repoDir, "config", "--add", "wit.add.hook", "npm run lint")
	testutil.RunGit(t, repoDir, "config", "--add", "wit.add.hook", "npm test")

	cfg, err := sync.Load(context.Background(), git.NewRunner(repoDir))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	assertConfiguredModes(t, cfg)
	assertConfiguredPaths(t, cfg)
}

func assertConfiguredModes(t *testing.T, cfg sync.Config) {
	t.Helper()

	if cfg.IgnoredDefault != sync.ModeSymlink {
		t.Fatalf("IgnoredDefault = %q, want %q", cfg.IgnoredDefault, sync.ModeSymlink)
	}

	if cfg.UntrackedDefault != sync.ModeCopy {
		t.Fatalf("UntrackedDefault = %q, want %q", cfg.UntrackedDefault, sync.ModeCopy)
	}
}

func assertConfiguredPaths(t *testing.T, cfg sync.Config) {
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

	if len(cfg.AddHooks) != 2 || cfg.AddHooks[0] != "npm run lint" || cfg.AddHooks[1] != "npm test" {
		t.Fatalf("AddHooks = %#v", cfg.AddHooks)
	}
}
