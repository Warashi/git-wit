package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Warashi/git-wit/internal/wit/integrate"
	"github.com/spf13/cobra"
)

var (
	errForceWithMerged          = errors.New("--force cannot be combined with --merged")
	errMergedRemovalRequiresYes = errors.New("merged removal requires --yes when stdin is not interactive")
	errMergedWithID             = errors.New("--merged accepts no worktree id")
	errYesWithoutMerged         = errors.New("--yes requires --merged")
)

func newRemoveCommand(deps dependencies) *cobra.Command {
	//nolint:exhaustruct // Cobra commands are configured field-by-field for readability.
	cmd := &cobra.Command{}
	cmd.Use = "rm <id>"
	cmd.Short = "Remove a managed worktree"
	cmd.ValidArgsFunction = completeManagedIDs(deps)

	merged := false
	yes := false
	force := false

	cmd.Flags().BoolVar(&merged, "merged", false, "remove all safely integrated managed worktrees")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation when removing merged worktrees")
	cmd.Flags().BoolVar(&force, "force", false, "remove the worktree even when it contains untracked or modified files")
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		return validateRemoveArgs(cmd, args, merged, yes, force)
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cwd, err := deps.cwd()
		if err != nil {
			return fmt.Errorf("get cwd: %w", err)
		}

		if merged {
			return runMergedRemoval(cmd, cwd, yes)
		}

		err = integrate.Remove(cmd.Context(), cwd, args[0], force, cmd.OutOrStdout(), cmd.ErrOrStderr())
		if err != nil {
			return fmt.Errorf("run rm: %w", err)
		}

		return nil
	}

	return cmd
}

func validateRemoveArgs(cmd *cobra.Command, args []string, merged bool, yes bool, force bool) error {
	if merged {
		if len(args) != 0 {
			return errMergedWithID
		}

		if force {
			return errForceWithMerged
		}

		return nil
	}

	if yes {
		return errYesWithoutMerged
	}

	return cobra.ExactArgs(1)(cmd, args)
}

func runMergedRemoval(cmd *cobra.Command, cwd string, yes bool) error {
	candidates, err := integrate.MergedCandidates(cmd.Context(), cwd)
	if err != nil {
		return fmt.Errorf("find merged worktrees: %w", err)
	}

	err = writeMergedCandidates(cmd, candidates)
	if err != nil {
		return err
	}

	if len(candidates) == 0 {
		return nil
	}

	proceed, err := proceedWithMergedRemoval(cmd, yes)
	if err != nil || !proceed {
		return err
	}

	results := integrate.RemoveMerged(cmd.Context(), cwd, candidates, nil)

	return writeMergedRemovalResults(cmd, results)
}

func writeMergedCandidates(cmd *cobra.Command, candidates []integrate.Candidate) error {
	for _, candidate := range candidates {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "merged\t%s\t%s\n", candidate.ID, candidate.Path)
		if err != nil {
			return fmt.Errorf("write candidate: %w", err)
		}
	}

	return nil
}

func proceedWithMergedRemoval(cmd *cobra.Command, yes bool) (bool, error) {
	if yes {
		return true, nil
	}

	return confirmMergedRemoval(cmd)
}

func writeMergedRemovalResults(cmd *cobra.Command, results []integrate.RemovalResult) error {
	failures := make([]error, 0, len(results))

	for _, result := range results {
		if result.Err == nil {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "removed\t%s\n", result.Candidate.ID)
			if err != nil {
				return fmt.Errorf("write removal result: %w", err)
			}

			continue
		}

		// Skipped candidates were deliberately left in place; report them
		// on stdout and keep the exit code clean.
		if result.Skipped {
			_, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"skipped\t%s\t%s\n",
				result.Candidate.ID,
				singleLineError(result.Err),
			)
			if err != nil {
				return fmt.Errorf("write skipped result: %w", err)
			}

			continue
		}

		_, err := fmt.Fprintf(
			cmd.ErrOrStderr(),
			"failed\t%s\t%s\n",
			result.Candidate.ID,
			singleLineError(result.Err),
		)
		if err != nil {
			return fmt.Errorf("write removal failure: %w", err)
		}

		failures = append(failures, fmt.Errorf("%s: %w", result.Candidate.ID, result.Err))
	}

	if len(failures) != 0 {
		return fmt.Errorf("remove merged worktrees: %w", errors.Join(failures...))
	}

	return nil
}

func confirmMergedRemoval(cmd *cobra.Command) (bool, error) {
	if isNonInteractiveInput(cmd.InOrStdin()) {
		return false, errMergedRemovalRequiresYes
	}

	err := writePrompt(cmd)
	if err != nil {
		return false, err
	}

	return readConfirmation(cmd.InOrStdin())
}

func singleLineError(err error) string {
	replacer := strings.NewReplacer("\r", " ", "\n", " ", "\t", " ")

	return replacer.Replace(err.Error())
}
