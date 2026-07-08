package cli

import (
	"strings"

	"github.com/Warashi/git-wit/internal/wit/query"
	"github.com/spf13/cobra"
)

func completeManagedIDs(deps dependencies) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		cwd, err := deps.cwd()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		entries, err := query.List(cmd.Context(), cwd)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		completions := make([]cobra.Completion, 0, len(entries))
		for _, entry := range entries {
			if !strings.HasPrefix(entry.ID, toComplete) {
				continue
			}

			completions = append(completions, cobra.CompletionWithDesc(entry.ID, managedIDDescription(entry)))
		}

		return completions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
	}
}

func managedIDDescription(entry query.Entry) string {
	parts := make([]string, 0, 3)

	if memo := sanitizeCompletionText(entry.Memo); memo != "" {
		parts = append(parts, memo)
	}

	switch {
	case entry.Branch != "":
		parts = append(parts, sanitizeCompletionText(entry.Branch))
	case entry.Head != "":
		parts = append(parts, "HEAD "+sanitizeCompletionText(entry.Head))
	}

	if entry.PRNumber != "" {
		parts = append(parts, "PR #"+sanitizeCompletionText(entry.PRNumber))
	}

	if len(parts) == 0 {
		return "managed worktree"
	}

	return strings.Join(parts, " · ")
}

func sanitizeCompletionText(value string) string {
	replacer := strings.NewReplacer("\t", " ", "\n", " ", "\r", " ")

	return strings.TrimSpace(replacer.Replace(value))
}
