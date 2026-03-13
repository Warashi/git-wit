package repository_test

import (
	"context"
	"os"
	"testing"

	"github.com/Warashi/git-wit/internal/repository"
	"github.com/Warashi/git-wit/internal/testutil"
)

func TestDiscover(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)

	repo, err := repository.Discover(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if repo.Root() != repoDir {
		t.Fatalf("Root() = %q, want %q", repo.Root(), repoDir)
	}
}

func TestEnsureWorktreeRoot(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)

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
