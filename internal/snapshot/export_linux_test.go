//go:build linux

//nolint:testpackage // Test-only exports need access to unexported helpers.
package snapshot

import "io/fs"

func ShouldFallbackLinuxCloneErrorForTest(err error) bool {
	return shouldFallbackLinuxCloneError(err)
}

func TryCloneFileLinuxForTest(
	srcPath string,
	destPath string,
	mode fs.FileMode,
	clone func(destFD int, srcFD int) error,
) (bool, error) {
	return tryCloneFileLinux(srcPath, destPath, mode, clone)
}
