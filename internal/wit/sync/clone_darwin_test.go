//go:build darwin

package sync_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Warashi/git-wit/internal/wit/sync"
	"golang.org/x/sys/unix"
)

func TestTryClonePathDarwinForTest(t *testing.T) {
	t.Parallel()

	t.Run("success", testTryClonePathDarwinSuccess)
	t.Run("fallback", testTryClonePathDarwinFallback)
	t.Run("hard error", testTryClonePathDarwinHardError)
}

func TestShouldFallbackDarwinCloneErrorForTest(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "enosys", err: unix.ENOSYS, want: true},
		{name: "enotsup", err: unix.ENOTSUP, want: true},
		{name: "exdev", err: unix.EXDEV, want: true},
		{name: "eperm", err: unix.EPERM, want: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := sync.ShouldFallbackDarwinCloneErrorForTest(testCase.err)
			if got != testCase.want {
				t.Fatalf("ShouldFallbackDarwinCloneErrorForTest() = %t, want %t", got, testCase.want)
			}
		})
	}
}

func testTryClonePathDarwinSuccess(t *testing.T) {
	t.Parallel()

	srcPath := writeDarwinCloneSourceFile(t)
	destPath := filepath.Join(t.TempDir(), "dest.txt")

	handled, err := sync.TryClonePathDarwinForTest(
		srcPath,
		destPath,
		func(string, string, int) error {
			return nil
		},
	)
	if err != nil {
		t.Fatalf("TryClonePathDarwinForTest() error = %v", err)
	}

	if !handled {
		t.Fatal("TryClonePathDarwinForTest() handled = false, want true")
	}
}

func testTryClonePathDarwinFallback(t *testing.T) {
	t.Parallel()

	srcPath := writeDarwinCloneSourceFile(t)
	destPath := filepath.Join(t.TempDir(), "dest.txt")

	handled, err := sync.TryClonePathDarwinForTest(
		srcPath,
		destPath,
		func(string, string, int) error {
			return unix.EXDEV
		},
	)
	if err != nil {
		t.Fatalf("TryClonePathDarwinForTest() error = %v", err)
	}

	if handled {
		t.Fatal("TryClonePathDarwinForTest() handled = true, want false")
	}
}

func testTryClonePathDarwinHardError(t *testing.T) {
	t.Parallel()

	srcPath := writeDarwinCloneSourceFile(t)
	destPath := filepath.Join(t.TempDir(), "dest.txt")

	handled, err := sync.TryClonePathDarwinForTest(
		srcPath,
		destPath,
		func(string, string, int) error {
			return unix.EPERM
		},
	)
	if !errors.Is(err, unix.EPERM) {
		t.Fatalf("TryClonePathDarwinForTest() error = %v, want %v", err, unix.EPERM)
	}

	if handled {
		t.Fatal("TryClonePathDarwinForTest() handled = true, want false")
	}
}

func writeDarwinCloneSourceFile(t *testing.T) string {
	t.Helper()

	srcPath := filepath.Join(t.TempDir(), "src.txt")

	err := os.WriteFile(srcPath, []byte("hello\n"), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return srcPath
}
