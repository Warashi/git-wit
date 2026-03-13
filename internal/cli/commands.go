package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	addwt "github.com/Warashi/git-wit/internal/add"
	dirwt "github.com/Warashi/git-wit/internal/dir"
	lswt "github.com/Warashi/git-wit/internal/ls"
	mergewt "github.com/Warashi/git-wit/internal/merge"
	prunewt "github.com/Warashi/git-wit/internal/prune"
	rmwt "github.com/Warashi/git-wit/internal/rm"
	"github.com/spf13/cobra"
)

var errSystemPruneRequiresYes = errors.New("system prune requires --yes when stdin is not interactive")

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

	system := false
	yes := false

	cmd.Flags().BoolVar(&system, "system", false, "scan the managed worktree root for orphaned directories")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation when deleting orphaned directories")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		cwd, err := deps.cwd()
		if err != nil {
			return fmt.Errorf("get cwd: %w", err)
		}

		if system {
			return runSystemPrune(cmd, cwd, yes)
		}

		result, err := prunewt.Run(cmd.Context(), cwd)
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

func runSystemPrune(cmd *cobra.Command, cwd string, yes bool) error {
	result, err := prunewt.ScanSystem(cmd.Context(), cwd)
	if err != nil {
		return fmt.Errorf("run system prune: %w", err)
	}

	err = writeSystemPruneScan(cmd, result)
	if err != nil {
		return err
	}

	if len(result.OrphanDirs) == 0 {
		return nil
	}

	if !yes {
		ok, err := confirmSystemPrune(cmd)
		if err != nil {
			return err
		}

		if !ok {
			return nil
		}
	}

	removedDirs, err := prunewt.RemoveSystemOrphans(result.OrphanDirs)
	if err != nil {
		return fmt.Errorf("remove system orphan dirs: %w", err)
	}

	return writeRemovedDirs(cmd, removedDirs)
}

func writeSystemPruneScan(cmd *cobra.Command, result prunewt.Result) error {
	err := writeTaggedPaths(cmd, "orphan-dir", result.OrphanDirs)
	if err != nil {
		return err
	}

	return writeTaggedPaths(cmd, "broken-symlink", result.BrokenSymlinks)
}

func writeRemovedDirs(cmd *cobra.Command, removedDirs []string) error {
	return writeTaggedPaths(cmd, "removed-dir", removedDirs)
}

func writeTaggedPaths(cmd *cobra.Command, tag string, paths []string) error {
	var err error

	for _, path := range paths {
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", tag, path)
		if err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}

	return nil
}

func confirmSystemPrune(cmd *cobra.Command) (bool, error) {
	if isNonInteractiveInput(cmd.InOrStdin()) {
		return false, errSystemPruneRequiresYes
	}

	err := writePrompt(cmd)
	if err != nil {
		return false, err
	}

	return readConfirmation(cmd.InOrStdin())
}

func writePrompt(cmd *cobra.Command) error {
	_, err := fmt.Fprint(cmd.OutOrStdout(), "Proceed? [y/N] ")
	if err != nil {
		return fmt.Errorf("write prompt: %w", err)
	}

	return nil
}

func readConfirmation(input io.Reader) (bool, error) {
	reader := bufio.NewReader(input)

	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}

	answer := strings.TrimSpace(strings.ToLower(line))

	return answer == "y" || answer == "yes", nil
}

func isNonInteractiveInput(input io.Reader) bool {
	file, ok := input.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return true
	}

	return info.Mode()&os.ModeCharDevice == 0
}
