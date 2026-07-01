package query_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/Warashi/git-wit/internal/testutil"
	"github.com/Warashi/git-wit/internal/wit/create"
	"github.com/Warashi/git-wit/internal/wit/query"
)

var shortHashPattern = regexp.MustCompile(`^[0-9a-f]{4,40}$`)

func TestListReportsDetachedHeadWorktree(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	created, err := create.Create(context.Background(), repoDir, time.Unix(100, 0), "memo", nil, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	entries, err := query.List(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("List() = %d entries, want 1", len(entries))
	}

	entry := entries[0]
	if entry.ID != created.ID {
		t.Fatalf("entry.ID = %q, want %q", entry.ID, created.ID)
	}

	if entry.Branch != "" {
		t.Fatalf("entry.Branch = %q, want empty (detached HEAD)", entry.Branch)
	}

	if !shortHashPattern.MatchString(entry.Head) {
		t.Fatalf("entry.Head = %q, want a short commit hash", entry.Head)
	}

	if entry.PRNumber != "" {
		t.Fatalf("entry.PRNumber = %q, want empty", entry.PRNumber)
	}
}

func TestListReportsCheckedOutBranch(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	created, err := create.Create(context.Background(), repoDir, time.Unix(200, 0), "memo", nil, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	testutil.RunGit(t, created.Path, "checkout", "-b", "feature/example")

	entries, err := query.List(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("List() = %d entries, want 1", len(entries))
	}

	if got := entries[0].Branch; got != "feature/example" {
		t.Fatalf("entry.Branch = %q, want %q", got, "feature/example")
	}

	if !shortHashPattern.MatchString(entries[0].Head) {
		t.Fatalf("entry.Head = %q, want a short commit hash", entries[0].Head)
	}
}
