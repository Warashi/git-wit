//go:build !linux && !darwin

package cli

import "os"

// Platforms without a termios probe fall back to the char-device heuristic;
// it misclassifies null devices as terminals, but stays on the safe side for
// regular files and pipes.
func isTerminalFile(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}
