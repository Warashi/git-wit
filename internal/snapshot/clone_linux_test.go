//go:build linux

package snapshot_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Warashi/git-wit/internal/snapshot"
	"golang.org/x/sys/unix"
)

func TestTryCloneFileLinuxForTest(t *testing.T) {
	t.Parallel()

	t.Run("success", testTryCloneFileLinuxSuccess)
	t.Run("fallback", testTryCloneFileLinuxFallback)
	t.Run("hard error", testTryCloneFileLinuxHardError)
}

func TestShouldFallbackLinuxCloneErrorForTest(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "badf", err: unix.EBADF, want: true},
		{name: "inval", err: unix.EINVAL, want: true},
		{name: "opnotsupp", err: unix.EOPNOTSUPP, want: true},
		{name: "exdev", err: unix.EXDEV, want: true},
		{name: "enotty", err: unix.ENOTTY, want: true},
		{name: "eperm", err: unix.EPERM, want: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := snapshot.ShouldFallbackLinuxCloneErrorForTest(testCase.err)
			if got != testCase.want {
				t.Fatalf("ShouldFallbackLinuxCloneErrorForTest() = %t, want %t", got, testCase.want)
			}
		})
	}
}

func writeCloneSourceFile(t *testing.T) string {
	t.Helper()

	srcPath := filepath.Join(t.TempDir(), "src.txt")

	err := os.WriteFile(srcPath, []byte("hello\n"), fs.FileMode(0o600))
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return srcPath
}

func testTryCloneFileLinuxSuccess(t *testing.T) {
	t.Parallel()

	srcPath := writeCloneSourceFile(t)
	destPath := filepath.Join(t.TempDir(), "dest.txt")

	handled, err := snapshot.TryCloneFileLinuxForTest(
		srcPath,
		destPath,
		0o600,
		func(int, int) error {
			return nil
		},
	)
	if err != nil {
		t.Fatalf("TryCloneFileLinuxForTest() error = %v", err)
	}

	if !handled {
		t.Fatal("TryCloneFileLinuxForTest() handled = false, want true")
	}
}

func testTryCloneFileLinuxFallback(t *testing.T) {
	t.Parallel()

	srcPath := writeCloneSourceFile(t)
	destPath := filepath.Join(t.TempDir(), "dest.txt")

	handled, err := snapshot.TryCloneFileLinuxForTest(
		srcPath,
		destPath,
		0o600,
		func(int, int) error {
			return unix.EXDEV
		},
	)
	if err != nil {
		t.Fatalf("TryCloneFileLinuxForTest() error = %v", err)
	}

	if handled {
		t.Fatal("TryCloneFileLinuxForTest() handled = true, want false")
	}
}

func testTryCloneFileLinuxHardError(t *testing.T) {
	t.Parallel()

	srcPath := writeCloneSourceFile(t)
	destPath := filepath.Join(t.TempDir(), "dest.txt")

	handled, err := snapshot.TryCloneFileLinuxForTest(
		srcPath,
		destPath,
		0o600,
		func(int, int) error {
			return unix.EPERM
		},
	)
	if !errors.Is(err, unix.EPERM) {
		t.Fatalf("TryCloneFileLinuxForTest() error = %v, want %v", err, unix.EPERM)
	}

	if handled {
		t.Fatal("TryCloneFileLinuxForTest() handled = true, want false")
	}
}
