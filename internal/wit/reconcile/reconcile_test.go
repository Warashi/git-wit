package reconcile_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Warashi/git-wit/internal/wit/reconcile"
)

// A cancelled context makes every ownership probe fail; that must surface as
// an error instead of classifying the worktree as an orphan.
func TestResolveWorktreeOwnerPropagatesContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := reconcile.ResolveWorktreeOwner(ctx, t.TempDir())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ResolveWorktreeOwner() error = %v, want context.Canceled", err)
	}
}

func TestFindBrokenSymlinksToleratesUnreadableEntries(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	err := os.Symlink(filepath.Join(root, "missing-target"), filepath.Join(root, "broken-link"))
	if err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	sealedDir := filepath.Join(root, "sealed")

	err = os.MkdirAll(sealedDir, 0o750)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	err = os.Chmod(sealedDir, 0o000)
	if err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}

	t.Cleanup(func() {
		// #nosec G302 -- restore a writable test directory for cleanup.
		_ = os.Chmod(sealedDir, 0o700)
	})

	broken, err := reconcile.FindBrokenSymlinks(root)
	if err != nil {
		t.Fatalf("FindBrokenSymlinks() error = %v, want unreadable entries to be skipped", err)
	}

	want := filepath.Join(root, "broken-link")
	if len(broken) != 1 || broken[0] != want {
		t.Fatalf("FindBrokenSymlinks() = %#v, want [%q]", broken, want)
	}
}
