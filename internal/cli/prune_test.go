package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Warashi/git-wit/internal/testutil"
	"github.com/Warashi/git-wit/internal/wit/catalog"
)

func TestRootCommandPruneRejectsYesWithoutSystem(t *testing.T) {
	t.Parallel()

	cmd := newTestRootCommand(t.TempDir(), time.Unix(100, 0))
	cmd.SetArgs([]string{"prune", "--yes"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--yes requires --system") {
		t.Fatalf("Execute() error = %v, want --yes requires --system", err)
	}
}

func TestRootCommandSystemPruneReportsRemovedDirsOnPartialFailure(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	worktreeRoot := strings.TrimSpace(string(testutil.RunGit(t, repoDir, "config", "wit.worktree.root")))

	// Two orphan directories: removal proceeds in listing order, so the
	// first succeeds and the second fails on its read-only subdirectory.
	removableID := catalog.NewID()
	stuckID := catalog.NewID()
	removableDir := filepath.Join(worktreeRoot, removableID)
	stuckDir := filepath.Join(worktreeRoot, stuckID)

	mustMkdir(t, removableDir)
	mustMkdir(t, filepath.Join(stuckDir, "sealed"))
	writeFile(t, filepath.Join(stuckDir, "sealed"), "inner.txt", "stuck")

	// #nosec G302 -- the read-only directory is what makes the removal fail.
	err := os.Chmod(filepath.Join(stuckDir, "sealed"), 0o500)
	if err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}

	t.Cleanup(func() {
		// #nosec G302 -- restore a writable test directory for cleanup.
		_ = os.Chmod(filepath.Join(stuckDir, "sealed"), 0o700)
	})

	var stdout bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(100, 0))
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"prune", "--system", "--yes"})

	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "remove system orphan dirs") {
		t.Fatalf("Execute() error = %v, want removal failure", err)
	}

	if !strings.Contains(stdout.String(), "removed-dir\t"+removableDir+"\n") {
		t.Fatalf("stdout = %q, want removed-dir line for %q", stdout.String(), removableDir)
	}
}
