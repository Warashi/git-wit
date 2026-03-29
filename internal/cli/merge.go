package cli

import (
	"fmt"

	"github.com/Warashi/git-wit/internal/wit/integrate"
	"github.com/spf13/cobra"
)

func newMergeCommand(deps dependencies) *cobra.Command {
	//nolint:exhaustruct // Cobra commands are configured field-by-field for readability.
	cmd := &cobra.Command{}
	cmd.Use = "merge <id>"
	cmd.Short = "Merge a managed worktree"
	cmd.Args = cobra.ExactArgs(1)

	remove := false
	cmd.Flags().BoolVar(&remove, "rm", false, "remove the worktree after a successful merge")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cwd, err := deps.cwd()
		if err != nil {
			return fmt.Errorf("get cwd: %w", err)
		}

		err = integrate.Merge(cmd.Context(), cwd, args[0], remove, cmd.OutOrStdout(), cmd.ErrOrStderr())
		if err != nil {
			return fmt.Errorf("run merge: %w", err)
		}

		return nil
	}

	return cmd
}
