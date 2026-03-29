package cli

import (
	"fmt"

	"github.com/Warashi/git-wit/internal/wit/query"
	"github.com/spf13/cobra"
)

func newIDCommand(deps dependencies) *cobra.Command {
	//nolint:exhaustruct // Cobra commands are configured field-by-field for readability.
	cmd := &cobra.Command{}
	cmd.Use = "id"
	cmd.Short = "Print the current managed worktree ID"
	cmd.Args = cobra.NoArgs
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		cwd, err := deps.cwd()
		if err != nil {
			return fmt.Errorf("get cwd: %w", err)
		}

		worktreeID, err := query.CurrentID(cmd.Context(), cwd)
		if err != nil {
			return fmt.Errorf("run id: %w", err)
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), worktreeID)
		if err != nil {
			return fmt.Errorf("write output: %w", err)
		}

		return nil
	}

	return cmd
}
