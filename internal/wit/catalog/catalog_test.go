package catalog_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Warashi/git-wit/internal/testutil"
	"github.com/Warashi/git-wit/internal/wit/catalog"
)

func TestOpenAndEnsureWorktreeRoot(t *testing.T) {
	repoDir := testutil.InitGitRepo(t)
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	repo, err := catalog.Open(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	wantRoot := filepath.Join(dataHome, "git-wit", "worktrees")
	if repo.WorktreeRoot() != wantRoot {
		t.Fatalf("WorktreeRoot() = %q, want %q", repo.WorktreeRoot(), wantRoot)
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

func TestStoreLoadListDeleteExists(t *testing.T) {
	t.Parallel()

	repo := openRepo(t)
	first := catalog.NewRecord(time.Unix(100, 0), "first")
	second := catalog.NewRecord(time.Unix(200, 0), "second")

	storeRecord(t, repo, second)
	storeRecord(t, repo, first)
	assertLoad(t, repo, first)
	assertList(t, repo, first, second)
	assertExists(t, repo, first.ID, true)
	deleteRecord(t, repo, first.ID)
	assertExists(t, repo, first.ID, false)
}

func TestOpenRejectsRelativeWorktreeRoot(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	testutil.RunGit(t, repoDir, "config", "wit.worktree.root", "./relative-root")

	_, err := catalog.Open(context.Background(), repoDir)
	if err == nil {
		t.Fatal("Open() error = nil, want relative-root rejection")
	}

	if !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("Open() error = %v, want mention of absolute path", err)
	}
}

func TestOpenIgnoresRelativeXDGDataHome(t *testing.T) {
	repoDir := testutil.InitGitRepo(t)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_DATA_HOME", "relative/data-home")

	repo, err := catalog.Open(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	wantRoot := filepath.Join(homeDir, ".local", "share", "git-wit", "worktrees")
	if repo.WorktreeRoot() != wantRoot {
		t.Fatalf("WorktreeRoot() = %q, want %q", repo.WorktreeRoot(), wantRoot)
	}
}

func TestCurrentIDResolvesSymlinkedWorktreeRoot(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	baseDir := t.TempDir()
	realRoot := filepath.Join(baseDir, "real-root")
	linkRoot := filepath.Join(baseDir, "link-root")

	err := os.MkdirAll(realRoot, 0o750)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	err = os.Symlink(realRoot, linkRoot)
	if err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	testutil.RunGit(t, repoDir, "config", "wit.worktree.root", linkRoot)

	repo, err := catalog.Open(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	record := catalog.NewRecord(time.Unix(400, 0), "memo")
	storeRecord(t, repo, record)

	worktreePath := repo.WorktreePath(record.ID)
	testutil.RunGit(t, repoDir, "worktree", "add", "-d", worktreePath)

	worktreeRepo, err := catalog.Open(context.Background(), worktreePath)
	if err != nil {
		t.Fatalf("Open(worktree) error = %v", err)
	}

	got, err := worktreeRepo.CurrentID()
	if err != nil {
		t.Fatalf("CurrentID() error = %v", err)
	}

	if got != record.ID {
		t.Fatalf("CurrentID() = %q, want %q", got, record.ID)
	}
}

func TestListSkipsCorruptEntries(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)

	repo, err := catalog.Open(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	good := catalog.NewRecord(time.Unix(100, 0), "good")
	storeRecord(t, repo, good)

	// A foreign ref name, a non-JSON blob, and a blob whose id disagrees
	// with its ref name must all be skipped instead of failing List.
	storeRawRef(t, repoDir, "refs/git-wit/not-a-uuid", `{"id":"not-a-uuid"}`)
	storeRawRef(t, repoDir, "refs/git-wit/"+catalog.NewID(), "not json at all")
	storeRawRef(t, repoDir, "refs/git-wit/"+catalog.NewID(), `{"id":"`+good.ID+`","memo":"impostor"}`)

	assertList(t, repo, good)
}

func storeRawRef(t *testing.T, repoDir string, ref string, blobContents string) {
	t.Helper()

	hash := strings.TrimSpace(string(testutil.RunGitWithInput(t, repoDir, blobContents, "hash-object", "-w", "--stdin")))
	testutil.RunGit(t, repoDir, "update-ref", ref, hash)
}

func TestCurrentID(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	worktreeRoot := filepath.Join(t.TempDir(), "worktrees")
	testutil.RunGit(t, repoDir, "config", "wit.worktree.root", worktreeRoot)

	repo, err := catalog.Open(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	err = repo.EnsureWorktreeRoot()
	if err != nil {
		t.Fatalf("EnsureWorktreeRoot() error = %v", err)
	}

	record := catalog.NewRecord(time.Unix(300, 0), "memo")

	err = repo.Store(context.Background(), record)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	worktreePath := repo.WorktreePath(record.ID)
	testutil.RunGit(t, repoDir, "worktree", "add", "-d", worktreePath)

	nestedDir := filepath.Join(worktreePath, "nested")

	err = os.MkdirAll(nestedDir, 0o750)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	worktreeRepo, err := catalog.Open(context.Background(), nestedDir)
	if err != nil {
		t.Fatalf("Open(worktree) error = %v", err)
	}

	got, err := worktreeRepo.CurrentID()
	if err != nil {
		t.Fatalf("CurrentID() error = %v", err)
	}

	if got != record.ID {
		t.Fatalf("CurrentID() = %q, want %q", got, record.ID)
	}
}

func openRepo(t *testing.T) catalog.Repository {
	t.Helper()

	repoDir := testutil.InitGitRepo(t)

	repo, err := catalog.Open(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	return repo
}

func storeRecord(t *testing.T, repo catalog.Repository, record catalog.Record) {
	t.Helper()

	err := repo.Store(context.Background(), record)
	if err != nil {
		t.Fatalf("Store(%s) error = %v", record.ID, err)
	}
}

func assertLoad(t *testing.T, repo catalog.Repository, want catalog.Record) {
	t.Helper()

	got, err := repo.Load(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got != want {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func assertList(t *testing.T, repo catalog.Repository, want ...catalog.Record) {
	t.Helper()

	items, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(items) != len(want) {
		t.Fatalf("len(List()) = %d, want %d", len(items), len(want))
	}

	for index := range want {
		if items[index] != want[index] {
			t.Fatalf("List()[%d] = %#v, want %#v", index, items[index], want[index])
		}
	}
}

func deleteRecord(t *testing.T, repo catalog.Repository, id string) {
	t.Helper()

	err := repo.Delete(context.Background(), id)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func assertExists(t *testing.T, repo catalog.Repository, id string, want bool) {
	t.Helper()

	exists, err := repo.Exists(context.Background(), id)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if exists != want {
		t.Fatalf("Exists() = %t, want %t", exists, want)
	}
}
