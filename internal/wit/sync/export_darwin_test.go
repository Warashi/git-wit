//go:build darwin

//nolint:testpackage // Test-only exports need access to unexported helpers.
package sync

func ShouldFallbackDarwinCloneErrorForTest(err error) bool {
	return shouldFallbackDarwinCloneError(err)
}

func TryClonePathDarwinForTest(
	srcPath string,
	destPath string,
	clone func(src string, dst string, flags int) error,
) (bool, error) {
	return tryClonePathDarwin(srcPath, destPath, clone)
}
