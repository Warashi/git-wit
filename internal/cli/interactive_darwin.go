//go:build darwin

package cli

import (
	"math"
	"os"

	"golang.org/x/sys/unix"
)

func isTerminalFile(file *os.File) bool {
	fd := file.Fd()
	if fd > math.MaxInt {
		return false
	}

	_, err := unix.IoctlGetTermios(int(fd), unix.TIOCGETA)

	return err == nil
}
