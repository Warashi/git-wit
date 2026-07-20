package cli

import (
	"io"
	"os"
)

// isNonInteractiveInput reports whether input cannot answer a confirmation
// prompt. Non-file readers (e.g. buffers injected by tests) count as
// interactive; files must actually be terminals — a char-device check alone
// would let /dev/null pass as interactive and turn a required confirmation
// into a silent no-op.
func isNonInteractiveInput(input io.Reader) bool {
	file, ok := input.(*os.File)
	if !ok {
		return false
	}

	return !isTerminalFile(file)
}
