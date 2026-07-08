package cli

import (
	"fmt"

	"github.com/Warashi/git-wit/internal/wit/query"
	"github.com/spf13/cobra"
)

func newDirCommand(deps dependencies) *cobra.Command {
	//nolint:exhaustruct // Cobra commands are configured field-by-field for readability.
	cmd := &cobra.Command{}
	cmd.Use = "dir <id>"
	cmd.Short = "Print a managed worktree path"
	cmd.Args = cobra.ExactArgs(1)
	cmd.ValidArgsFunction = completeManagedIDs(deps)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cwd, err := deps.cwd()
		if err != nil {
			return fmt.Errorf("get cwd: %w", err)
		}

		worktreePath, err := query.Dir(cmd.Context(), cwd, args[0])
		if err != nil {
			return fmt.Errorf("run dir: %w", err)
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), worktreePath)
		if err != nil {
			return fmt.Errorf("write output: %w", err)
		}

		return nil
	}

	return cmd
}
