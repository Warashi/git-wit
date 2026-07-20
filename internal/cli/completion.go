package cli

import (
	"strings"

	"github.com/Warashi/git-wit/internal/wit/query"
	"github.com/spf13/cobra"
)

const managedIDDescriptionPartsCap = 2

func completeManagedIDs(deps dependencies) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// rm --merged accepts no worktree id, so offering one would only
		// complete an argument the command is guaranteed to reject.
		if flag := cmd.Flags().Lookup("merged"); flag != nil && flag.Changed {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		cwd, err := deps.cwd()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError | cobra.ShellCompDirectiveNoFileComp
		}

		entries, err := query.List(cmd.Context(), cwd, false)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError | cobra.ShellCompDirectiveNoFileComp
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
	parts := make([]string, 0, managedIDDescriptionPartsCap)

	if memo := sanitizeCompletionText(entry.Memo); memo != "" {
		parts = append(parts, memo)
	}

	// Branched worktrees already surface the branch name; detached worktrees use
	// the short HEAD hash instead so the description stays compact. PR info is
	// never resolved here — completion always lists without GitHub lookups.
	switch {
	case entry.Branch != "":
		parts = append(parts, sanitizeCompletionText(entry.Branch))
	case entry.Head != "":
		parts = append(parts, "HEAD "+sanitizeCompletionText(entry.Head))
	}

	if len(parts) == 0 {
		return "managed worktree"
	}

	return strings.Join(parts, " · ")
}

func sanitizeCompletionText(value string) string {
	return strings.TrimSpace(strings.NewReplacer("\t", " ", "\n", " ", "\r", " ").Replace(value))
}
