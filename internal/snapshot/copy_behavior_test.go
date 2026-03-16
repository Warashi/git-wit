package snapshot_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Warashi/git-wit/internal/snapshot"
)

func TestCopyFileFallsBackWhenCloneUnhandled(t *testing.T) {
	t.Parallel()

	srcPath := filepath.Join(t.TempDir(), "src.txt")
	destPath := filepath.Join(t.TempDir(), "dest.txt")

	err := os.WriteFile(srcPath, []byte("hello\n"), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	copier := snapshot.NewCopierForTest(
		func(string, string) (bool, error) {
			return false, nil
		},
		func(string, string, fs.FileMode) (bool, error) {
			return false, nil
		},
	)

	err = copier.CopyFile(srcPath, destPath, 0o600)
	if err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}

	// #nosec G304 -- test reads from the temporary destination path it created.
	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(got) != "hello\n" {
		t.Fatalf("copied contents = %q, want %q", string(got), "hello\n")
	}
}

func TestCopyFilePropagatesCloneError(t *testing.T) {
	t.Parallel()

	srcPath := filepath.Join(t.TempDir(), "src.txt")
	destPath := filepath.Join(t.TempDir(), "dest.txt")

	err := os.WriteFile(srcPath, []byte("hello\n"), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	copier := snapshot.NewCopierForTest(
		func(string, string) (bool, error) {
			return false, nil
		},
		func(string, string, fs.FileMode) (bool, error) {
			return false, errCloneFailed
		},
	)

	err = copier.CopyFile(srcPath, destPath, 0o600)
	if !errors.Is(err, errCloneFailed) {
		t.Fatalf("CopyFile() error = %v, want %v", err, errCloneFailed)
	}
}

func TestCopyDirFallsBackWhenCloneUnhandled(t *testing.T) {
	t.Parallel()

	srcRoot := t.TempDir()
	destRoot := t.TempDir()
	srcPath := filepath.Join(srcRoot, "src")
	destPath := filepath.Join(destRoot, "dest")

	err := os.MkdirAll(srcPath, 0o750)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	err = os.WriteFile(filepath.Join(srcPath, "child.txt"), []byte("hello\n"), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	copier := snapshot.NewCopierForTest(
		func(string, string) (bool, error) {
			return false, nil
		},
		func(string, string, fs.FileMode) (bool, error) {
			return false, nil
		},
	)

	err = copier.CopyDir(srcPath, destPath, fs.ModeDir|0o750)
	if err != nil {
		t.Fatalf("CopyDir() error = %v", err)
	}

	// #nosec G304 -- test reads from the temporary destination path it created.
	got, err := os.ReadFile(filepath.Join(destPath, "child.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(got) != "hello\n" {
		t.Fatalf("copied contents = %q, want %q", string(got), "hello\n")
	}
}

func TestCopyDirStopsAfterCloneHandled(t *testing.T) {
	t.Parallel()

	srcRoot := t.TempDir()
	destRoot := t.TempDir()
	srcPath := filepath.Join(srcRoot, "src")
	destPath := filepath.Join(destRoot, "dest")

	err := os.MkdirAll(srcPath, 0o750)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	err = os.WriteFile(filepath.Join(srcPath, "child.txt"), []byte("hello\n"), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	copier := snapshot.NewCopierForTest(
		func(string, string) (bool, error) {
			return true, os.MkdirAll(destPath, 0o750)
		},
		func(string, string, fs.FileMode) (bool, error) {
			return false, nil
		},
	)

	err = copier.CopyDir(srcPath, destPath, fs.ModeDir|0o750)
	if err != nil {
		t.Fatalf("CopyDir() error = %v", err)
	}

	_, err = os.Stat(filepath.Join(destPath, "child.txt"))
	if !os.IsNotExist(err) {
		t.Fatalf("child.txt stat error = %v, want not exist", err)
	}
}

var errCloneFailed = errors.New("clone failed")
