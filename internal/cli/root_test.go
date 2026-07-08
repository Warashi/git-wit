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

	want := []string{"add", "ls", "id", "dir", "rm", "merge", "prune"}
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

	var stderr bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(100, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
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
	stderr.Reset()

	cmd = newTestRootCommand(repoDir, time.Unix(100, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"dir", worktreeID})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if strings.TrimSpace(stdout.String()) != worktreePath {
		t.Fatalf("dir output = %q, want %q", stdout.String(), worktreePath)
	}
}

func TestRootCommand_AddRoutesChildOutputToStderr(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)
	testutil.RunGit(t, repoDir, "config", "--add", "wit.add.hook", `printf hook-stdout`)
	testutil.RunGit(t, repoDir, "config", "--add", "wit.add.hook", `printf hook-stderr >&2`)

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(125, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"add", "memo"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	fields := strings.Split(strings.TrimSpace(stdout.String()), "\t")
	if len(fields) != 2 {
		t.Fatalf("add output = %q", stdout.String())
	}

	if !strings.Contains(stderr.String(), "hook-stdout") {
		t.Fatalf("stderr = %q, want hook stdout", stderr.String())
	}

	if !strings.Contains(stderr.String(), "hook-stderr") {
		t.Fatalf("stderr = %q, want hook stderr", stderr.String())
	}
}

func TestRootCommand_ID(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	worktreeID, worktreePath := addWorktree(t, repoDir, "memo")
	nestedDir := filepath.Join(worktreePath, "nested")
	mustMkdir(t, nestedDir)

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	cmd := newTestRootCommand(nestedDir, time.Unix(150, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"id"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if strings.TrimSpace(stdout.String()) != worktreeID {
		t.Fatalf("id output = %q, want %q", stdout.String(), worktreeID)
	}
}

func TestRootCommand_ListRemoveAndPrune(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	worktreeID, _ := addWorktree(t, repoDir, "memo")

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(200, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"ls"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	line := strings.TrimSuffix(stdout.String(), "\n")

	fields := strings.Split(line, "\t")
	if len(fields) != 7 {
		t.Fatalf("ls output = %q, want 7 tab-separated fields", stdout.String())
	}

	if fields[3] != "memo" {
		t.Fatalf("ls memo field = %q, want %q", fields[3], "memo")
	}

	// A freshly created worktree is a detached HEAD with no branch and no
	// pull request, but must report its HEAD commit.
	if fields[4] != "-" {
		t.Fatalf("ls branch field = %q, want %q", fields[4], "-")
	}

	if fields[5] == "" || fields[5] == "-" {
		t.Fatalf("ls head field = %q, want a commit hash", fields[5])
	}

	if fields[6] != "-" {
		t.Fatalf("ls pr field = %q, want %q", fields[6], "-")
	}

	stdout.Reset()
	stderr.Reset()

	cmd = newTestRootCommand(repoDir, time.Unix(200, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"rm", worktreeID})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()

	cmd = newTestRootCommand(repoDir, time.Unix(200, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
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

	var stderr bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(300, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"merge", "--rm", worktreeID})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	testutil.RunGit(t, repoDir, "update-ref", "-d", "refs/git-wit/"+worktreeID)

	orphanID, _ := addWorktree(t, repoDir, "orphan")
	testutil.RunGit(t, repoDir, "update-ref", "-d", "refs/git-wit/"+orphanID)

	_, brokenPath := addWorktree(t, repoDir, "broken")

	err = os.Symlink("missing-target", filepath.Join(brokenPath, "link"))
	if err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()

	cmd = newTestRootCommand(repoDir, time.Unix(300, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
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

func TestRootCommand_MergePassesConflictOutputToStdout(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	worktreeID, worktreePath := addWorktree(t, repoDir, "memo")
	writeFile(t, repoDir, "shared.txt", "main\n")
	testutil.RunGit(t, repoDir, "add", "shared.txt")
	testutil.RunGit(t, repoDir, "commit", "-m", "main change")

	writeFile(t, worktreePath, "shared.txt", "worktree\n")
	testutil.RunGit(t, worktreePath, "add", "shared.txt")
	testutil.RunGit(t, worktreePath, "commit", "-m", "worktree change")

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(350, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"merge", worktreeID})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}

	if !strings.Contains(stdout.String(), "CONFLICT") {
		t.Fatalf("stdout = %q, want conflict output", stdout.String())
	}
}

func TestRootCommand_SystemPruneRemovesUnknownOrphans(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(400, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"add", "memo"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	fields := strings.Split(strings.TrimSpace(stdout.String()), "\t")
	if len(fields) != 2 {
		t.Fatalf("add output = %q", stdout.String())
	}

	worktreePath := fields[1]

	err = os.Remove(filepath.Join(worktreePath, ".git"))
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()

	cmd = newTestRootCommand(repoDir, time.Unix(400, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"prune", "--system", "--yes"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "removed-dir\t") {
		t.Fatalf("prune --system output = %q", stdout.String())
	}

	_, err = os.Stat(worktreePath)
	if !os.IsNotExist(err) {
		t.Fatalf("Stat() error = %v, want not exist", err)
	}
}

func TestRootCommand_SystemPruneLeavesRepoOwnedWorktreesUntouched(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	worktreeID, worktreePath := addWorktree(t, repoDir, "memo")
	testutil.RunGit(t, repoDir, "update-ref", "-d", "refs/git-wit/"+worktreeID)

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(500, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"prune", "--system", "--yes"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if strings.Contains(stdout.String(), "removed-dir\t") {
		t.Fatalf("prune --system output = %q", stdout.String())
	}

	_, err = os.Stat(worktreePath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}

func TestRootCommand_SystemPruneRequiresYesForNonInteractiveInput(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	_, worktreePath := addWorktree(t, repoDir, "memo")

	err := os.Remove(filepath.Join(worktreePath, ".git"))
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	inputFilePath := filepath.Join(t.TempDir(), "stdin.txt")

	err = os.WriteFile(inputFilePath, []byte(""), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// #nosec G304 -- test opens a file it created under t.TempDir.
	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	defer func() {
		closeErr := inputFile.Close()
		if closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	}()

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(600, 0))
	cmd.SetIn(inputFile)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"prune", "--system"})

	err = cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "requires --yes") {
		t.Fatalf("Execute() error = %v", err)
	}

	_, err = os.Stat(worktreePath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}

func TestRootCommand_IDArgumentCompletion(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	detachedID, _ := addWorktree(t, repoDir, "detached memo")
	branchedID, branchedPath := addWorktree(t, repoDir, "feature memo")
	testutil.RunGit(t, branchedPath, "checkout", "-b", "feature/example")

	for _, subcommand := range []string{"dir", "rm", "merge"} {
		output := runCompletion(t, repoDir, subcommand, "")

		if !strings.Contains(output, detachedID+"\tdetached memo · HEAD ") {
			t.Fatalf("%s completion output = %q, want detached worktree description", subcommand, output)
		}

		if !strings.Contains(output, branchedID+"\tfeature memo · feature/example") {
			t.Fatalf("%s completion output = %q, want branched worktree description", subcommand, output)
		}

		if !strings.Contains(output, ":36\n") {
			t.Fatalf("%s completion output = %q, want no-file-completion directive", subcommand, output)
		}
	}
}

func TestRootCommand_IDArgumentCompletionFiltersByPrefix(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	matchedID, _ := addWorktree(t, repoDir, "matched memo")
	otherID, _ := addWorktree(t, repoDir, "other memo")

	output := runCompletion(t, repoDir, "dir", distinctPrefix(matchedID, otherID))

	if !strings.Contains(output, matchedID+"\tmatched memo") {
		t.Fatalf("completion output = %q, want matching id", output)
	}

	if strings.Contains(output, otherID+"\tother memo") {
		t.Fatalf("completion output = %q, got unexpected id", output)
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

	var stderr bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(100, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
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

func runCompletion(t *testing.T, repoDir string, subcommand string, toComplete string) string {
	t.Helper()

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(700, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{cobra.ShellCompRequestCmd, subcommand, toComplete})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v, stderr = %q", err, stderr.String())
	}

	return stdout.String()
}

func distinctPrefix(first string, second string) string {
	if first == second {
		return ""
	}

	maxPrefix := len(first)
	if len(second) < maxPrefix {
		maxPrefix = len(second)
	}

	for i := 1; i <= maxPrefix; i++ {
		if first[i-1] != second[i-1] {
			return first[:i]
		}
	}

	if len(first) > maxPrefix {
		return first[:maxPrefix+1]
	}

	return ""
}

func writeFile(t *testing.T, dirPath string, name string, contents string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dirPath, name), []byte(contents), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
