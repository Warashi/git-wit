package cli

import (
	"time"

	"github.com/spf13/cobra"
)

func NewRootCommandForTest(cwd func() (string, error), now func() time.Time) *cobra.Command {
	return newRootCommand(dependencies{
		cwd: cwd,
		now: now,
	})
}
