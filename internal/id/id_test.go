package id_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Warashi/git-wit/internal/add"
	"github.com/Warashi/git-wit/internal/id"
	"github.com/Warashi/git-wit/internal/testutil"
)

func TestRunReturnsManagedWorktreeIDFromSubdirectory(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	result, err := add.Run(context.Background(), repoDir, time.Unix(100, 0), "memo", nil, nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	nestedDir := filepath.Join(result.Path, "nested")
	mustMkdir(t, nestedDir)

	got, err := id.Run(context.Background(), nestedDir)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got != result.ID {
		t.Fatalf("Run() = %q, want %q", got, result.ID)
	}
}

func TestRunFailsOutsideManagedWorktree(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	_, err := id.Run(context.Background(), repoDir)
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "not in a managed worktree") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunFailsForOrphanedWorktree(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	result, err := add.Run(context.Background(), repoDir, time.Unix(200, 0), "memo", nil, nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	testutil.RunGit(t, repoDir, "update-ref", "-d", "refs/git-wit/"+result.ID)

	_, err = id.Run(context.Background(), result.Path)
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "not in a managed worktree") {
		t.Fatalf("Run() error = %v", err)
	}
}

func configureWorktreeRoot(t *testing.T, repoDir string) {
	t.Helper()

	testutil.RunGit(t, repoDir, "config", "wit.worktree.root", filepath.Join(t.TempDir(), "worktrees"))
}

func mustMkdir(t *testing.T, dirPath string) {
	t.Helper()

	err := os.MkdirAll(dirPath, 0o750)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
}
