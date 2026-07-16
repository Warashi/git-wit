package integrate_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Warashi/git-wit/internal/testutil"
	"github.com/Warashi/git-wit/internal/wit/catalog"
	"github.com/Warashi/git-wit/internal/wit/create"
	"github.com/Warashi/git-wit/internal/wit/integrate"
)

func TestMergedCandidatesAreSortedAndExcludeCurrentWorktree(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	newer := createWorktree(t, repoDir, time.Unix(200, 0))
	older := createWorktree(t, repoDir, time.Unix(100, 0))

	candidates, err := integrate.MergedCandidates(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("MergedCandidates() error = %v", err)
	}

	assertCandidateIDs(t, candidates, older.ID, newer.ID)

	candidates, err = integrate.MergedCandidates(context.Background(), older.Path)
	if err != nil {
		t.Fatalf("MergedCandidates() from managed worktree error = %v", err)
	}

	assertCandidateIDs(t, candidates, newer.ID)
}

func TestRemoveMergedContinuesAfterDirtyWorktree(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	dirty := createWorktree(t, repoDir, time.Unix(100, 0))
	clean := createWorktree(t, repoDir, time.Unix(200, 0))

	err := os.WriteFile(filepath.Join(dirty.Path, "dirty.txt"), []byte("dirty\n"), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	candidates, err := integrate.MergedCandidates(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("MergedCandidates() error = %v", err)
	}

	results := integrate.RemoveMerged(context.Background(), repoDir, candidates, nil)
	if len(results) != 2 {
		t.Fatalf("len(RemoveMerged()) = %d, want 2", len(results))
	}

	if results[0].Candidate.ID != dirty.ID || results[0].Err == nil {
		t.Fatalf("RemoveMerged()[0] = %#v, want dirty worktree failure", results[0])
	}

	if results[1].Candidate.ID != clean.ID || results[1].Err != nil {
		t.Fatalf("RemoveMerged()[1] = %#v, want clean worktree success", results[1])
	}

	assertMetadataExists(t, repoDir, dirty.ID, true)
	assertMetadataExists(t, repoDir, clean.ID, false)
}

func TestRemoveMergedRevalidatesCandidate(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	created := createWorktree(t, repoDir, time.Unix(100, 0))

	candidates, err := integrate.MergedCandidates(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("MergedCandidates() error = %v", err)
	}

	testutil.RunGit(t, created.Path, "commit", "--allow-empty", "-m", "post-confirmation change")

	results := integrate.RemoveMerged(context.Background(), repoDir, candidates, nil)
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("RemoveMerged() = %#v, want one revalidation failure", results)
	}

	assertMetadataExists(t, repoDir, created.ID, true)
}

func createWorktree(t *testing.T, repoDir string, now time.Time) create.Result {
	t.Helper()

	result, err := create.Create(context.Background(), repoDir, now, "memo", nil, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	return result
}

func configureWorktreeRoot(t *testing.T, repoDir string) {
	t.Helper()

	testutil.RunGit(t, repoDir, "config", "wit.worktree.root", filepath.Join(t.TempDir(), "worktrees"))
}

func assertCandidateIDs(t *testing.T, candidates []integrate.Candidate, want ...string) {
	t.Helper()

	if len(candidates) != len(want) {
		t.Fatalf("len(candidates) = %d, want %d", len(candidates), len(want))
	}

	for index := range want {
		if candidates[index].ID != want[index] {
			t.Fatalf("candidates[%d].ID = %q, want %q", index, candidates[index].ID, want[index])
		}
	}
}

func assertMetadataExists(t *testing.T, repoDir string, worktreeID string, want bool) {
	t.Helper()

	repo, err := catalog.Open(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	got, err := repo.Exists(context.Background(), worktreeID)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if got != want {
		t.Fatalf("Exists() = %t, want %t", got, want)
	}
}
