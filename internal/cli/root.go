// Package cli wires the Cobra command tree.
package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

var errNotImplemented = errors.New("subcommand is not implemented")

// NewRootCommand builds the root git-wit command.
func NewRootCommand() *cobra.Command {
	//nolint:exhaustruct // Cobra commands are configured field-by-field for readability.
	cmd := &cobra.Command{}
	cmd.Use = "git-wit"
	cmd.Short = "Manage Git worktrees by stable IDs and metadata"
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	cmd.AddCommand(
		newStubCommand("add", "Create a managed worktree"),
		newStubCommand("ls", "List managed worktrees"),
		newStubCommand("dir", "Print a managed worktree path"),
		newStubCommand("rm", "Remove a managed worktree"),
		newStubCommand("merge", "Merge a managed worktree"),
		newStubCommand("prune", "Repair orphaned git-wit state"),
	)

	return cmd
}

func newStubCommand(use string, short string) *cobra.Command {
	//nolint:exhaustruct // Cobra commands are configured field-by-field for readability.
	cmd := &cobra.Command{}
	cmd.Use = use
	cmd.Short = short
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		return errNotImplemented
	}

	return cmd
}
