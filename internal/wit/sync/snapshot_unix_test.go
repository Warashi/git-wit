//go:build unix

package sync_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Warashi/git-wit/internal/git"
	"github.com/Warashi/git-wit/internal/testutil"
	"github.com/Warashi/git-wit/internal/wit/sync"
	"golang.org/x/sys/unix"
)

// Copying an ignored directory that contains a FIFO must skip the FIFO
// instead of opening it, which would block until a writer appears.
func TestApplySkipsIrregularFiles(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	destDir := t.TempDir()

	writeFile(t, filepath.Join(repoDir, ".gitignore"), "data/\n")
	testutil.RunGit(t, repoDir, "add", ".gitignore")
	testutil.RunGit(t, repoDir, "commit", "-m", "add ignore rules")

	mustMkdir(t, filepath.Join(repoDir, "data"))
	writeFile(t, filepath.Join(repoDir, "data", "regular.txt"), "kept")

	err := unix.Mkfifo(filepath.Join(repoDir, "data", "pipe"), 0o600)
	if err != nil {
		t.Fatalf("Mkfifo() error = %v", err)
	}

	//nolint:exhaustruct // Test input only sets fields relevant to resolution.
	cfg := sync.Config{
		IgnoredDefault:   sync.ModeCopy,
		UntrackedDefault: sync.ModeNone,
	}

	err = sync.Apply(context.Background(), git.NewRunner(repoDir), repoDir, destDir, cfg)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	assertFileContents(t, filepath.Join(destDir, "data", "regular.txt"), "kept")

	_, err = os.Lstat(filepath.Join(destDir, "data", "pipe"))
	if !os.IsNotExist(err) {
		t.Fatalf("Lstat(pipe) error = %v, want not exist", err)
	}
}
