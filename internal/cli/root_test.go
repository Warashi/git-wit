package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Warashi/git-wit/internal/cli"
	"github.com/Warashi/git-wit/internal/testutil"
	"github.com/spf13/cobra"
)

func TestNewRootCommand_HasSubcommands(t *testing.T) {
	t.Parallel()

	cmd := cli.NewRootCommand()

	want := []string{"add", "ls", "dir", "rm", "merge", "prune"}
	for _, name := range want {
		got, _, err := cmd.Find([]string{name})
		if err != nil || got == cmd {
			t.Fatalf("Find(%q) error = %v, got root = %t", name, err, got == cmd)
		}
	}
}

func TestRootCommand_AddAndDir(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	var stdout bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(100, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"add", "memo"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	fields := strings.Split(strings.TrimSpace(stdout.String()), "\t")
	if len(fields) != 2 {
		t.Fatalf("add output = %q", stdout.String())
	}

	worktreeID := fields[0]
	worktreePath := fields[1]

	stdout.Reset()

	cmd = newTestRootCommand(repoDir, time.Unix(100, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"dir", worktreeID})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if strings.TrimSpace(stdout.String()) != worktreePath {
		t.Fatalf("dir output = %q, want %q", stdout.String(), worktreePath)
	}
}

func TestRootCommand_ListRemoveAndPrune(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	worktreeID, _ := addWorktree(t, repoDir, "memo")

	var stdout bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(200, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"ls"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "\tmemo\n") {
		t.Fatalf("ls output = %q", stdout.String())
	}

	stdout.Reset()

	cmd = newTestRootCommand(repoDir, time.Unix(200, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"rm", worktreeID})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	stdout.Reset()

	cmd = newTestRootCommand(repoDir, time.Unix(200, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"prune"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if stdout.String() != "" {
		t.Fatalf("prune output = %q, want empty", stdout.String())
	}
}

func TestRootCommand_MergeAndPruneWarnings(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	worktreeID, worktreePath := addWorktree(t, repoDir, "memo")
	writeFile(t, worktreePath, "feature.txt", "feature\n")
	testutil.RunGit(t, worktreePath, "add", "feature.txt")
	testutil.RunGit(t, worktreePath, "commit", "-m", "feature")

	var stdout bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(300, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"merge", "--rm", worktreeID})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	testutil.RunGit(t, repoDir, "update-ref", "-d", "refs/git-wit/"+worktreeID)

	worktreeRoot := filepath.Dir(worktreePath)
	mustMkdir(t, filepath.Join(worktreeRoot, "orphan"))
	writeFile(t, filepath.Join(worktreeRoot, "orphan"), "note", "orphan\n")

	_, brokenPath := addWorktree(t, repoDir, "broken")

	err = os.Symlink("missing-target", filepath.Join(brokenPath, "link"))
	if err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	stdout.Reset()

	cmd = newTestRootCommand(repoDir, time.Unix(300, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"prune"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "orphan-dir\t") {
		t.Fatalf("prune output = %q", stdout.String())
	}

	if !strings.Contains(stdout.String(), "broken-symlink\t") {
		t.Fatalf("prune output = %q", stdout.String())
	}
}

func newTestRootCommand(repoDir string, now time.Time) *cobra.Command {
	return cli.NewRootCommandForTest(
		func() (string, error) {
			return repoDir, nil
		},
		func() time.Time {
			return now
		},
	)
}

func addWorktree(t *testing.T, repoDir string, memo string) (string, string) {
	t.Helper()

	var stdout bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(100, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"add", memo})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	fields := strings.Split(strings.TrimSpace(stdout.String()), "\t")
	if len(fields) != 2 {
		t.Fatalf("add output = %q", stdout.String())
	}

	return fields[0], fields[1]
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

func writeFile(t *testing.T, dirPath string, name string, contents string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dirPath, name), []byte(contents), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
