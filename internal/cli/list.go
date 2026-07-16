package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Warashi/git-wit/internal/wit/query"
	"github.com/spf13/cobra"
)

func newListCommand(deps dependencies) *cobra.Command {
	var outputJSON bool

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

		if outputJSON {
			err = writeListJSON(cmd.OutOrStdout(), entries)
			if err != nil {
				return fmt.Errorf("write JSON output: %w", err)
			}

			return nil
		}

		for _, entry := range entries {
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				entry.ID,
				entry.CreatedAt.Format(time.RFC3339),
				entry.Path,
				entry.Memo,
				displayOrDash(entry.Branch),
				displayOrDash(entry.Head),
				prDisplay(entry.PRNumber),
				displayOrDash(entry.State),
			)
			if err != nil {
				return fmt.Errorf("write output: %w", err)
			}
		}

		return nil
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "output managed worktrees as JSON")

	return cmd
}

type listJSONEntry struct {
	ID         string       `json:"id"`
	CreatedAt  time.Time    `json:"created_at"` //nolint:tagliatelle // Keep the CLI schema consistent with metadata JSON.
	Path       string       `json:"path"`
	Memo       string       `json:"memo"`
	Branch     *string      `json:"branch"`
	Head       *string      `json:"head"`
	PRNumber   *json.Number `json:"pr_number"` //nolint:tagliatelle // The public CLI schema uses snake_case.
	State      *string      `json:"state"`
	Integrated bool         `json:"integrated"`
}

func writeListJSON(writer io.Writer, entries []query.Entry) error {
	output := make([]listJSONEntry, 0, len(entries))
	for _, entry := range entries {
		output = append(output, listJSONEntry{
			ID:         entry.ID,
			CreatedAt:  entry.CreatedAt,
			Path:       entry.Path,
			Memo:       entry.Memo,
			Branch:     optionalString(entry.Branch),
			Head:       optionalString(entry.Head),
			PRNumber:   optionalJSONNumber(entry.PRNumber),
			State:      optionalString(entry.State),
			Integrated: entry.Integrated,
		})
	}

	if err := json.NewEncoder(writer).Encode(output); err != nil {
		return fmt.Errorf("encode entries: %w", err)
	}

	return nil
}

func optionalJSONNumber(value string) *json.Number {
	if value == "" {
		return nil
	}

	number := json.Number(value)

	return &number
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

const noValuePlaceholder = "-"

// displayOrDash renders a placeholder for empty fields (e.g. detached HEAD
// has no branch) so column alignment stays predictable for tab-separated
// output.
func displayOrDash(value string) string {
	if value == "" {
		return noValuePlaceholder
	}

	return value
}

// prDisplay renders a pull request number with a leading "#", or a
// placeholder when no pull request could be resolved.
func prDisplay(number string) string {
	if number == "" {
		return noValuePlaceholder
	}

	return "#" + number
}
