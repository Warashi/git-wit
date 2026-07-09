package cli

import (
	"fmt"

	"github.com/Warashi/git-wit/internal/wit/query"
	"github.com/spf13/cobra"
)

func newMemoCommand(deps dependencies) *cobra.Command {
	//nolint:exhaustruct // Cobra commands are configured field-by-field for readability.
	cmd := &cobra.Command{}
	cmd.Use = "memo [id]"
	cmd.Short = "Print a managed worktree memo"
	cmd.Args = cobra.MaximumNArgs(1)
	cmd.ValidArgsFunction = completeManagedIDs(deps)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cwd, err := deps.cwd()
		if err != nil {
			return fmt.Errorf("get cwd: %w", err)
		}

		var worktreeID string
		if len(args) > 0 {
			worktreeID = args[0]
		}

		memo, err := query.Memo(cmd.Context(), cwd, worktreeID)
		if err != nil {
			return fmt.Errorf("run memo: %w", err)
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), memo)
		if err != nil {
			return fmt.Errorf("write output: %w", err)
		}

		return nil
	}

	return cmd
}
