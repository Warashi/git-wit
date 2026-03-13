package repository_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Warashi/git-wit/internal/repository"
	"github.com/Warashi/git-wit/internal/testutil"
)

func TestDiscover(t *testing.T) {
	repoDir := testutil.InitGitRepo(t)
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	repo, err := repository.Discover(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if repo.Root() != repoDir {
		t.Fatalf("Root() = %q, want %q", repo.Root(), repoDir)
	}

	wantRoot := filepath.Join(dataHome, "git-wit", "worktrees")
	if repo.WorktreeRoot() != wantRoot {
		t.Fatalf("WorktreeRoot() = %q, want %q", repo.WorktreeRoot(), wantRoot)
	}
}

func TestEnsureWorktreeRoot(t *testing.T) {
	repoDir := testutil.InitGitRepo(t)
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	repo, err := repository.Discover(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	err = repo.EnsureWorktreeRoot()
	if err != nil {
		t.Fatalf("EnsureWorktreeRoot() error = %v", err)
	}

	_, err = os.Stat(repo.WorktreeRoot())
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}
