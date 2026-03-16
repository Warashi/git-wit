//go:build linux

package snapshot

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"

	"golang.org/x/sys/unix"
)

type linuxCloneFileFunc func(destFD int, srcFD int) error

var errFileDescriptorOverflow = errors.New("file descriptor overflows int")

func tryCloneDir(string, string) (bool, error) {
	return false, nil
}

func tryCloneFile(srcPath string, destPath string, mode fs.FileMode) (bool, error) {
	return tryCloneFileLinux(srcPath, destPath, mode, unix.IoctlFileClone)
}

func tryCloneFileLinux(
	srcPath string,
	destPath string,
	mode fs.FileMode,
	clone linuxCloneFileFunc,
) (bool, error) {
	// #nosec G304 -- source paths are discovered from the active repository.
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return false, fmt.Errorf("open source file: %w", err)
	}

	defer func() {
		_ = srcFile.Close()
	}()

	// #nosec G304 -- destination paths are under the managed worktree root.
	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return false, fmt.Errorf("open destination file: %w", err)
	}

	defer func() {
		_ = destFile.Close()
	}()

	destFD, err := fileDescriptor(destFile)
	if err != nil {
		return false, err
	}

	srcFD, err := fileDescriptor(srcFile)
	if err != nil {
		return false, err
	}

	err = clone(destFD, srcFD)
	if err == nil {
		return true, nil
	}

	if shouldFallbackLinuxCloneError(err) {
		return false, nil
	}

	return false, err
}

func shouldFallbackLinuxCloneError(err error) bool {
	return errors.Is(err, unix.EBADF) ||
		errors.Is(err, unix.EINVAL) ||
		errors.Is(err, unix.EOPNOTSUPP) ||
		errors.Is(err, unix.EXDEV) ||
		errors.Is(err, unix.ENOTTY)
}

func fileDescriptor(file *os.File) (int, error) {
	fd := file.Fd()
	if fd > math.MaxInt {
		return 0, fmt.Errorf("%w: %d", errFileDescriptorOverflow, fd)
	}

	return int(fd), nil
}
