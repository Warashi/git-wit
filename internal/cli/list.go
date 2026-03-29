package cli

import (
	"fmt"
	"time"

	"github.com/Warashi/git-wit/internal/wit/query"
	"github.com/spf13/cobra"
)

func newListCommand(deps dependencies) *cobra.Command {
	//nolint:exhaustruct // Cobra commands are configured field-by-field for readability.
	cmd := &cobra.Command{}
	cmd.Use = "ls"
	cmd.Short = "List managed worktrees"
	cmd.Args = cobra.NoArgs
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		cwd, err := deps.cwd()
		if err != nil {
			return fmt.Errorf("get cwd: %w", err)
		}

		entries, err := query.List(cmd.Context(), cwd)
		if err != nil {
			return fmt.Errorf("run ls: %w", err)
		}

		for _, entry := range entries {
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"%s\t%s\t%s\t%s\n",
				entry.ID,
				entry.CreatedAt.Format(time.RFC3339),
				entry.Path,
				entry.Memo,
			)
			if err != nil {
				return fmt.Errorf("write output: %w", err)
			}
		}

		return nil
	}

	return cmd
}
