package metadata_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Warashi/git-wit/internal/git"
	"github.com/Warashi/git-wit/internal/metadata"
	"github.com/Warashi/git-wit/internal/testutil"
)

func TestStoreAndLoad(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	runner := git.NewRunner(repoDir)
	item := metadata.New("01ARZ3NDEKTSV4RRFFQ69G5FAV", time.Unix(100, 0), "memo")

	err := metadata.Store(context.Background(), runner, item)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	got, err := metadata.Load(context.Background(), runner, item.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got != item {
		t.Fatalf("Load() = %#v, want %#v", got, item)
	}
}

func TestList(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	runner := git.NewRunner(repoDir)
	items := []metadata.Metadata{
		metadata.New("01ARZ3NDEKTSV4RRFFQ69G5FAV", time.Unix(200, 0), "second"),
		metadata.New("01ARZ3NDEKTSV4RRFFQ69G5FAW", time.Unix(100, 0), "first"),
	}

	for _, item := range items {
		err := metadata.Store(context.Background(), runner, item)
		if err != nil {
			t.Fatalf("Store() error = %v", err)
		}
	}

	got, err := metadata.List(context.Background(), runner)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(got) != len(items) {
		t.Fatalf("len(List()) = %d, want %d", len(got), len(items))
	}

	if got[0].ID != items[1].ID || got[1].ID != items[0].ID {
		t.Fatalf("List() order = %#v", got)
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	runner := git.NewRunner(repoDir)
	item := metadata.New("01ARZ3NDEKTSV4RRFFQ69G5FAV", time.Unix(100, 0), "memo")

	err := metadata.Store(context.Background(), runner, item)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	err = metadata.Delete(context.Background(), runner, item.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	exists, err := metadata.Exists(context.Background(), runner, item.ID)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if exists {
		t.Fatal("Exists() = true, want false")
	}
}

func TestLoad_IgnoresUnknownFields(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	runner := git.NewRunner(repoDir)
	worktreeID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	raw := fmt.Sprintf(
		`{"id":"%s","created_at":"1970-01-01T00:01:40Z","memo":"memo","version":"1.0","extra":"ignored"}`,
		worktreeID,
	)

	hash := string(testutil.RunGitWithInput(t, repoDir, raw, "hash-object", "-w", "--stdin"))
	testutil.RunGit(t, repoDir, "update-ref", "refs/git-wit/"+worktreeID, strings.TrimSpace(hash))

	got, err := metadata.Load(context.Background(), runner, worktreeID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.ID != worktreeID || got.Memo != "memo" || got.Version != "1.0" {
		t.Fatalf("Load() = %#v", got)
	}
}
