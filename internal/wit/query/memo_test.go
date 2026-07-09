package query_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Warashi/git-wit/internal/testutil"
	"github.com/Warashi/git-wit/internal/wit/create"
	"github.com/Warashi/git-wit/internal/wit/query"
)

func TestMemoReturnsMemoForID(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	created, err := create.Create(context.Background(), repoDir, time.Unix(100, 0), "hello world", nil, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := query.Memo(context.Background(), repoDir, created.ID)
	if err != nil {
		t.Fatalf("Memo() error = %v", err)
	}

	if got != "hello world" {
		t.Fatalf("Memo() = %q, want %q", got, "hello world")
	}
}

func TestMemoReturnsMemoForCurrentWorktreeWhenIDEmpty(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	created, err := create.Create(context.Background(), repoDir, time.Unix(100, 0), "current worktree memo", nil, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := query.Memo(context.Background(), created.Path, "")
	if err != nil {
		t.Fatalf("Memo() error = %v", err)
	}

	if got != "current worktree memo" {
		t.Fatalf("Memo() = %q, want %q", got, "current worktree memo")
	}
}

func TestMemoFailsOutsideManagedWorktreeWhenIDEmpty(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	_, err := query.Memo(context.Background(), repoDir, "")
	if err == nil {
		t.Fatal("Memo() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "not in a managed worktree") {
		t.Fatalf("Memo() error = %v", err)
	}
}

func TestMemoFailsForOrphanedCurrentWorktreeWhenIDEmpty(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	created, err := create.Create(context.Background(), repoDir, time.Unix(100, 0), "memo", nil, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	testutil.RunGit(t, repoDir, "update-ref", "-d", "refs/git-wit/"+created.ID)

	_, err = query.Memo(context.Background(), created.Path, "")
	if err == nil {
		t.Fatal("Memo() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "not in a managed worktree") {
		t.Fatalf("Memo() error = %v, want 'not in a managed worktree'", err)
	}
}

func TestMemoFailsForUnknownID(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	_, err := query.Memo(context.Background(), repoDir, "0195e4d1-3d44-7a52-8e18-5f7b3c3d9a01")
	if err == nil {
		t.Fatal("Memo() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "unknown worktree id") {
		t.Fatalf("Memo() error = %v", err)
	}
}

func TestMemoFailsForInvalidID(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	_, err := query.Memo(context.Background(), repoDir, "not-a-uuid")
	if err == nil {
		t.Fatal("Memo() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "validate id") {
		t.Fatalf("Memo() error = %v", err)
	}
}
