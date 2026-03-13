package worktree_test

import (
	"testing"

	"github.com/Warashi/git-wit/internal/worktree"
)

func TestNewID(t *testing.T) {
	t.Parallel()

	id := worktree.NewID()

	err := worktree.ValidateID(id)
	if err != nil {
		t.Fatalf("ValidateID() error = %v", err)
	}
}

func TestPath(t *testing.T) {
	t.Parallel()

	repoRoot := "/tmp/repo"
	id := "01ARZ3NDEKTSV4RRFFQ69G5FAV"

	got := worktree.Path(repoRoot, id)

	want := "/tmp/repo/.git-wit/worktrees/01ARZ3NDEKTSV4RRFFQ69G5FAV"
	if got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}
