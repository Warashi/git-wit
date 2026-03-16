//go:build !linux && !darwin

package snapshot

import "io/fs"

func tryCloneDir(string, string) (bool, error) {
	return false, nil
}

func tryCloneFile(string, string, fs.FileMode) (bool, error) {
	return false, nil
}
