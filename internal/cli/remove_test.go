package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Warashi/git-wit/internal/testutil"
	"github.com/Warashi/git-wit/internal/wit/catalog"
	"github.com/Warashi/git-wit/internal/wit/create"
)

func TestRootCommandRemoveMergedContinuesAfterFailure(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	dirty := createManagedWorktree(t, repoDir, time.Unix(100, 0))
	clean := createManagedWorktree(t, repoDir, time.Unix(200, 0))
	unmerged := createManagedWorktree(t, repoDir, time.Unix(300, 0))

	integrateManagedWorktree(t, repoDir, dirty)
	integrateManagedWorktree(t, repoDir, clean)

	writeFile(t, dirty.Path, "dirty.txt", "dirty\n")
	testutil.RunGit(t, unmerged.Path, "commit", "--allow-empty", "-m", "unmerged")

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	cmd := newTestRootCommand(repoDir, time.Unix(400, 0))
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"rm", "--merged", "--yes"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want partial failure")
	}

	wantOutput := strings.Join([]string{
		"merged\t" + dirty.ID + "\t" + dirty.Path,
		"merged\t" + clean.ID + "\t" + clean.Path,
		"removed\t" + clean.ID,
		"",
	}, "\n")
	if stdout.String() != wantOutput {
		t.Fatalf("stdout = %q, want %q", stdout.String(), wantOutput)
	}

	if !strings.Contains(stderr.String(), "failed\t"+dirty.ID+"\t") {
		t.Fatalf("stderr = %q, want tagged dirty worktree failure", stderr.String())
	}

	assertManagedMetadata(t, repoDir, dirty.ID, true)
	assertManagedMetadata(t, repoDir, clean.ID, false)
	assertManagedMetadata(t, repoDir, unmerged.ID, true)
}

func TestRootCommandRemoveMergedCanBeCancelled(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)
	created := createManagedWorktree(t, repoDir, time.Unix(100, 0))
	integrateManagedWorktree(t, repoDir, created)

	var stdout bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(200, 0))
	cmd.SetIn(strings.NewReader("\n"))
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"rm", "--merged"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.HasSuffix(stdout.String(), "Proceed? [y/N] ") {
		t.Fatalf("stdout = %q, want confirmation prompt", stdout.String())
	}

	assertManagedMetadata(t, repoDir, created.ID, true)
}

func TestRootCommandRemoveMergedRequiresYesForNonInteractiveInput(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)
	created := createManagedWorktree(t, repoDir, time.Unix(100, 0))
	integrateManagedWorktree(t, repoDir, created)

	inputPath := filepath.Join(t.TempDir(), "stdin")

	err := os.WriteFile(inputPath, nil, 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// #nosec G304 -- test opens a file it created under t.TempDir.
	input, err := os.Open(inputPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	t.Cleanup(func() {
		if closeErr := input.Close(); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})

	var stdout bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(200, 0))
	cmd.SetIn(input)
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"rm", "--merged"})

	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "requires --yes") {
		t.Fatalf("Execute() error = %v, want requires --yes", err)
	}

	if !strings.Contains(stdout.String(), "merged\t"+created.ID+"\t") {
		t.Fatalf("stdout = %q, want candidate before error", stdout.String())
	}

	assertManagedMetadata(t, repoDir, created.ID, true)
}

func TestRootCommandRemoveMergedExcludesCurrentManagedWorktree(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)
	created := createManagedWorktree(t, repoDir, time.Unix(100, 0))
	integrateManagedWorktree(t, repoDir, created)

	var stdout bytes.Buffer

	cmd := newTestRootCommand(created.Path, time.Unix(200, 0))
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"rm", "--merged", "--yes"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	assertManagedMetadata(t, repoDir, created.ID, true)
}

func TestRootCommandRemoveRejectsInvalidFlagCombinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "merged with id",
			args: []string{"rm", "--merged", "01900000-0000-7000-8000-000000000000"},
			want: "--merged accepts no worktree id",
		},
		{
			name: "yes with id",
			args: []string{"rm", "--yes", "01900000-0000-7000-8000-000000000000"},
			want: "--yes requires --merged",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cmd := newTestRootCommand(t.TempDir(), time.Unix(100, 0))
			cmd.SetArgs(test.args)

			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Execute() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func createManagedWorktree(t *testing.T, repoDir string, now time.Time) create.Result {
	t.Helper()

	result, err := create.Create(context.Background(), repoDir, now, "memo", nil, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	return result
}

// integrateManagedWorktree commits work in the worktree and merges it into
// the repository, making the worktree a safe rm --merged candidate.
func integrateManagedWorktree(t *testing.T, repoDir string, created create.Result) {
	t.Helper()

	testutil.RunGit(t, created.Path, "commit", "--allow-empty", "-m", "work")
	head := strings.TrimSpace(string(testutil.RunGit(t, created.Path, "rev-parse", "HEAD")))
	testutil.RunGit(t, repoDir, "merge", head)
}

func assertManagedMetadata(t *testing.T, repoDir string, worktreeID string, want bool) {
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
