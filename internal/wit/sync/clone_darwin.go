//go:build darwin

package sync

import (
	"errors"
	"io/fs"

	"golang.org/x/sys/unix"
)

type darwinClonePathFunc func(src string, dst string, flags int) error

func tryCloneDir(srcPath string, destPath string) (bool, error) {
	return tryClonePathDarwin(srcPath, destPath, unix.Clonefile)
}

func tryCloneFile(srcPath string, destPath string, _ fs.FileMode) (bool, error) {
	return tryClonePathDarwin(srcPath, destPath, unix.Clonefile)
}

func tryClonePathDarwin(srcPath string, destPath string, clone darwinClonePathFunc) (bool, error) {
	err := clone(srcPath, destPath, 0)
	if err == nil {
		return true, nil
	}

	if shouldFallbackDarwinCloneError(err) {
		return false, nil
	}

	return false, err
}

func shouldFallbackDarwinCloneError(err error) bool {
	return errors.Is(err, unix.ENOSYS) ||
		errors.Is(err, unix.ENOTSUP) ||
		errors.Is(err, unix.EXDEV)
}
