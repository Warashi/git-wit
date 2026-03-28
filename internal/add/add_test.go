package add_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

	result, err := add.Run(context.Background(), repoDir, time.Unix(100, 0), "memo", nil, nil)
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

func TestRunExecutesAddHooksInWorkingDirectory(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	worktreeRoot := filepath.Join(t.TempDir(), "worktrees")
	testutil.RunGit(t, repoDir, "config", "wit.worktree.root", worktreeRoot)
	testutil.RunGit(t, repoDir, "config", "--add", "wit.add.hook", `pwd > hook-cwd.txt`)
	testutil.RunGit(t, repoDir, "config", "--add", "wit.add.hook", `printf second > hook-second.txt`)

	result, err := add.Run(context.Background(), repoDir, time.Unix(200, 0), "memo", nil, nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// #nosec G304 -- test reads from the temporary destination directory it created.
	got, err := os.ReadFile(filepath.Join(result.Path, "hook-cwd.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if strings.TrimSpace(string(got)) != result.Path {
		t.Fatalf("hook cwd = %q, want %q", strings.TrimSpace(string(got)), result.Path)
	}

	_, err = os.Stat(filepath.Join(result.Path, "hook-second.txt"))
	if err != nil {
		t.Fatalf("Stat() error = %v, want file to exist", err)
	}
}

func TestRunFailsWhenAddHookFails(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	worktreeRoot := filepath.Join(t.TempDir(), "worktrees")
	testutil.RunGit(t, repoDir, "config", "wit.worktree.root", worktreeRoot)
	testutil.RunGit(t, repoDir, "config", "--add", "wit.add.hook", `printf hooked > hook-ran.txt; exit 1`)

	_, err := add.Run(context.Background(), repoDir, time.Unix(300, 0), "memo", nil, nil)
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}

	entries, err := os.ReadDir(worktreeRoot)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("len(ReadDir()) = %d, want 1", len(entries))
	}

	worktreePath := filepath.Join(worktreeRoot, entries[0].Name())

	// #nosec G304 -- test reads from the temporary destination directory it created.
	got, err := os.ReadFile(filepath.Join(worktreePath, "hook-ran.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(got) != "hooked" {
		t.Fatalf("hook output = %q, want %q", string(got), "hooked")
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
