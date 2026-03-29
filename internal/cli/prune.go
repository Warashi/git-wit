package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Warashi/git-wit/internal/wit/reconcile"
	"github.com/spf13/cobra"
)

var errSystemPruneRequiresYes = errors.New("system prune requires --yes when stdin is not interactive")

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

		result, err := reconcile.Prune(cmd.Context(), cwd)
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
	result, err := reconcile.ScanSystem(cmd.Context(), cwd)
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

	removedDirs, err := reconcile.RemoveSystemOrphans(result.OrphanDirs)
	if err != nil {
		return fmt.Errorf("remove system orphan dirs: %w", err)
	}

	return writeRemovedDirs(cmd, removedDirs)
}

func writeSystemPruneScan(cmd *cobra.Command, result reconcile.Result) error {
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
