package cli

import (
	"fmt"

	"github.com/Warashi/git-wit/internal/wit/integrate"
	"github.com/spf13/cobra"
)

func newRemoveCommand(deps dependencies) *cobra.Command {
	//nolint:exhaustruct // Cobra commands are configured field-by-field for readability.
	cmd := &cobra.Command{}
	cmd.Use = "rm <id>"
	cmd.Short = "Remove a managed worktree"
	cmd.Args = cobra.ExactArgs(1)
	cmd.ValidArgsFunction = completeManagedIDs(deps)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cwd, err := deps.cwd()
		if err != nil {
			return fmt.Errorf("get cwd: %w", err)
		}

		err = integrate.Remove(cmd.Context(), cwd, args[0], cmd.OutOrStdout(), cmd.ErrOrStderr())
		if err != nil {
			return fmt.Errorf("run rm: %w", err)
		}

		return nil
	}

	return cmd
}
