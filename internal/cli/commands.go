package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	addwt "github.com/Warashi/git-wit/internal/add"
	dirwt "github.com/Warashi/git-wit/internal/dir"
	lswt "github.com/Warashi/git-wit/internal/ls"
	mergewt "github.com/Warashi/git-wit/internal/merge"
	prunewt "github.com/Warashi/git-wit/internal/prune"
	rmwt "github.com/Warashi/git-wit/internal/rm"
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
		newDirCommand(deps),
		newRemoveCommand(deps),
		newMergeCommand(deps),
		newPruneCommand(deps),
	}
}

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

		result, err := addwt.Run(cmd.Context(), cwd, deps.now(), args[0])
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

		entries, err := lswt.Run(cmd.Context(), cwd)
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

func newDirCommand(deps dependencies) *cobra.Command {
	//nolint:exhaustruct // Cobra commands are configured field-by-field for readability.
	cmd := &cobra.Command{}
	cmd.Use = "dir <id>"
	cmd.Short = "Print a managed worktree path"
	cmd.Args = cobra.ExactArgs(1)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cwd, err := deps.cwd()
		if err != nil {
			return fmt.Errorf("get cwd: %w", err)
		}

		worktreePath, err := dirwt.Run(cmd.Context(), cwd, args[0])
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

func newRemoveCommand(deps dependencies) *cobra.Command {
	//nolint:exhaustruct // Cobra commands are configured field-by-field for readability.
	cmd := &cobra.Command{}
	cmd.Use = "rm <id>"
	cmd.Short = "Remove a managed worktree"
	cmd.Args = cobra.ExactArgs(1)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cwd, err := deps.cwd()
		if err != nil {
			return fmt.Errorf("get cwd: %w", err)
		}

		err = rmwt.Run(cmd.Context(), cwd, args[0])
		if err != nil {
			return fmt.Errorf("run rm: %w", err)
		}

		return nil
	}

	return cmd
}

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

		err = mergewt.Run(cmd.Context(), cwd, args[0], remove)
		if err != nil {
			return fmt.Errorf("run merge: %w", err)
		}

		return nil
	}

	return cmd
}

func newPruneCommand(deps dependencies) *cobra.Command {
	//nolint:exhaustruct // Cobra commands are configured field-by-field for readability.
	cmd := &cobra.Command{}
	cmd.Use = "prune"
	cmd.Short = "Repair orphaned git-wit state"
	cmd.Args = cobra.NoArgs
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		cwd, err := deps.cwd()
		if err != nil {
			return fmt.Errorf("get cwd: %w", err)
		}

		result, err := prunewt.Run(context.Background(), cwd)
		if err != nil {
			return fmt.Errorf("run prune: %w", err)
		}

		for _, id := range result.RemovedRefs {
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "removed-ref\t%s\n", id)
			if err != nil {
				return fmt.Errorf("write output: %w", err)
			}
		}

		for _, dirPath := range result.OrphanDirs {
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "orphan-dir\t%s\n", dirPath)
			if err != nil {
				return fmt.Errorf("write output: %w", err)
			}
		}

		for _, linkPath := range result.BrokenSymlinks {
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "broken-symlink\t%s\n", linkPath)
			if err != nil {
				return fmt.Errorf("write output: %w", err)
			}
		}

		return nil
	}

	return cmd
}
