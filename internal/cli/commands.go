package cli

import (
	"os"
	"time"

	"github.com/spf13/cobra"
)

type dependencies struct {
	cwd func() (string, error)
	now func() time.Time
}

func defaultDependencies() dependencies {
	return dependencies{
		cwd: os.Getwd,
		now: time.Now,
	}
}

func newCommandTree(deps dependencies) []*cobra.Command {
	return []*cobra.Command{
		newAddCommand(deps),
		newListCommand(deps),
		newIDCommand(deps),
		newDirCommand(deps),
		newRemoveCommand(deps),
		newMergeCommand(deps),
		newPruneCommand(deps),
	}
}
