package add_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Warashi/git-wit/internal/add"
	"github.com/Warashi/git-wit/internal/testutil"
)

func TestRunSkipsIgnoredAndUntrackedByDefault(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	worktreeRoot := filepath.Join(t.TempDir(), "worktrees")
	testutil.RunGit(t, repoDir, "config", "wit.worktree.root", worktreeRoot)

	writeFile(t, filepath.Join(repoDir, ".gitignore"), "ignored/\n")
	testutil.RunGit(t, repoDir, "add", ".gitignore")
	testutil.RunGit(t, repoDir, "commit", "-m", "add ignore rules")

	mustMkdir(t, filepath.Join(repoDir, "ignored"))
	writeFile(t, filepath.Join(repoDir, "ignored", "a.txt"), "ignored")
	writeFile(t, filepath.Join(repoDir, "scratch.txt"), "scratch")

	result, err := add.Run(context.Background(), repoDir, time.Unix(100, 0), "memo")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	_, err = os.Stat(filepath.Join(result.Path, "ignored"))
	if !os.IsNotExist(err) {
		t.Fatalf("ignored stat error = %v, want not exist", err)
	}

	_, err = os.Stat(filepath.Join(result.Path, "scratch.txt"))
	if !os.IsNotExist(err) {
		t.Fatalf("scratch.txt stat error = %v, want not exist", err)
	}
}

func mustMkdir(t *testing.T, dirPath string) {
	t.Helper()

	err := os.MkdirAll(dirPath, 0o750)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
}

func writeFile(t *testing.T, filePath string, contents string) {
	t.Helper()

	err := os.WriteFile(filePath, []byte(contents), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
