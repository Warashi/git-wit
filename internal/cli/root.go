// Package cli wires the Cobra command tree.
package cli

import "github.com/spf13/cobra"

// NewRootCommand builds the root git-wit command.
func NewRootCommand() *cobra.Command {
	return newRootCommand(defaultDependencies())
}

func newRootCommand(deps dependencies) *cobra.Command {
	//nolint:exhaustruct // Cobra commands are configured field-by-field for readability.
	cmd := &cobra.Command{}
	cmd.Use = "git-wit"
	cmd.Short = "Manage Git worktrees by stable IDs and metadata"
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	cmd.AddCommand(newCommandTree(deps)...)

	return cmd
}
