package query

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Warashi/git-wit/internal/git"
	"github.com/Warashi/git-wit/internal/wit/catalog"
)

const ghExecutable = "gh"

// Entry is one row of ls output.
type Entry struct {
	ID        string
	CreatedAt time.Time
	Path      string
	Memo      string
	Branch    string
	Head      string
	PRNumber  string
}

// List lists managed worktrees.
func List(ctx context.Context, cwd string) ([]Entry, error) {
	repo, err := catalog.Open(ctx, cwd)
	if err != nil {
		return nil, fmt.Errorf("discover repository: %w", err)
	}

	items, err := repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list metadata: %w", err)
	}

	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		path := repo.WorktreePath(item.ID)
		branch, head := worktreeState(ctx, path)

		entries = append(entries, Entry{
			ID:        item.ID,
			CreatedAt: item.CreatedAt,
			Path:      path,
			Memo:      item.Memo,
			Branch:    branch,
			Head:      head,
			PRNumber:  pullRequestNumber(ctx, path, branch),
		})
	}

	return entries, nil
}

// worktreeState reports the current branch (empty when detached) and the
// short HEAD commit hash for the worktree at path. Errors are ignored: a
// worktree that is missing or otherwise unreadable simply reports empty
// values, since ls is a best-effort, read-only view.
func worktreeState(ctx context.Context, path string) (branch string, head string) {
	runner := git.NewRunner(path)

	if result, err := runner.Run(ctx, "symbolic-ref", "--short", "-q", "HEAD"); err == nil {
		branch = result.Stdout
	}

	if result, err := runner.Run(ctx, "rev-parse", "--short", "HEAD"); err == nil {
		head = result.Stdout
	}

	return branch, head
}

// pullRequestNumber returns the number of the pull request associated with
// branch, resolved via the optional `gh` CLI. It returns an empty string
// when gh is unavailable, the worktree is detached, or no PR is found, so
// ls remains usable without any GitHub integration configured.
func pullRequestNumber(ctx context.Context, path string, branch string) string {
	if branch == "" {
		return ""
	}

	// gh interprets a leading "-" as a flag rather than a branch name; git
	// itself refuses to create such branches, but guard defensively since
	// branch is passed straight through as a positional argument below.
	if strings.HasPrefix(branch, "-") {
		return ""
	}

	if _, err := exec.LookPath(ghExecutable); err != nil {
		return ""
	}

	// #nosec G204 -- gh is invoked directly via exec (no shell), so branch cannot inject
	// additional shell commands. The only remaining risk is gh mistaking branch for a flag,
	// which the leading-"-" check above already rules out.
	cmd := exec.CommandContext(ctx, ghExecutable, "pr", "view", branch, "--json", "number", "--jq", ".number")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}
