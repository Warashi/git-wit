package cli

import (
	"fmt"

	"github.com/Warashi/git-wit/internal/wit/create"
	"github.com/spf13/cobra"
)

func newAddCommand(deps dependencies) *cobra.Command {
	//nolint:exhaustruct // Cobra commands are configured field-by-field for readability.
	cmd := &cobra.Command{}
	cmd.Use = "add <memo>"
	cmd.Short = "Create a managed worktree"
	cmd.Args = cobra.ExactArgs(1)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cwd, err := deps.cwd()
		if err != nil {
			return fmt.Errorf("get cwd: %w", err)
		}

		result, err := create.Create(cmd.Context(), cwd, deps.now(), args[0], cmd.OutOrStdout(), cmd.ErrOrStderr())
		if err != nil {
			return fmt.Errorf("run add: %w", err)
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", result.ID, result.Path)
		if err != nil {
			return fmt.Errorf("write output: %w", err)
		}

		return nil
	}

	return cmd
}
