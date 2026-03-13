package worktree_test

import (
	"testing"

	"github.com/Warashi/git-wit/internal/worktree"
	"github.com/google/uuid"
)

func TestNewID(t *testing.T) {
	t.Parallel()

	worktreeID := worktree.NewID()

	err := worktree.ValidateID(worktreeID)
	if err != nil {
		t.Fatalf("ValidateID() error = %v", err)
	}

	parsed, err := uuid.Parse(worktreeID)
	if err != nil {
		t.Fatalf("uuid.Parse() error = %v", err)
	}

	if parsed.Version() != 7 {
		t.Fatalf("Version() = %d, want 7", parsed.Version())
	}
}

func TestValidateIDRejectsULID(t *testing.T) {
	t.Parallel()

	err := worktree.ValidateID("01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if err == nil {
		t.Fatal("ValidateID() error = nil, want error")
	}
}

func TestValidateIDRejectsNonV7UUID(t *testing.T) {
	t.Parallel()

	err := worktree.ValidateID("550e8400-e29b-41d4-a716-446655440000")
	if err == nil {
		t.Fatal("ValidateID() error = nil, want error")
	}
}

func TestPath(t *testing.T) {
	t.Parallel()

	repoRoot := "/tmp/repo"
	id := "0195e4d1-3d44-7a52-8e18-5f7b3c3d9a01"

	got := worktree.Path(repoRoot, id)

	want := "/tmp/.repo-git-wit/worktrees/0195e4d1-3d44-7a52-8e18-5f7b3c3d9a01"
	if got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}
