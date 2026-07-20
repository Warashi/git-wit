package query

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Warashi/git-wit/internal/git"
	"github.com/Warashi/git-wit/internal/wit/catalog"
)

const (
	ghExecutable = "gh"
	mergedState  = "Merged"
)

// Entry is one row of ls output.
type Entry struct {
	ID         string
	CreatedAt  time.Time
	Path       string
	Memo       string
	Branch     string
	Head       string
	PRNumber   string
	State      string
	Integrated bool
}

// List lists managed worktrees. withGitHub controls whether pull request
// information is resolved via the optional `gh` CLI; callers that only need
// local state (e.g. shell completion) should pass false to avoid the
// associated GitHub API calls.
func List(ctx context.Context, cwd string, withGitHub bool) ([]Entry, error) {
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
		entries = append(entries, buildEntry(ctx, cwd, repo, item, withGitHub))
	}

	return entries, nil
}

// Get returns the current read model for one managed worktree. See List for
// the meaning of withGitHub.
func Get(ctx context.Context, cwd string, worktreeID string, withGitHub bool) (Entry, error) {
	err := catalog.ValidateID(worktreeID)
	if err != nil {
		return Entry{}, fmt.Errorf("validate id: %w", err)
	}

	repo, err := catalog.Open(ctx, cwd)
	if err != nil {
		return Entry{}, fmt.Errorf("discover repository: %w", err)
	}

	exists, err := repo.Exists(ctx, worktreeID)
	if err != nil {
		return Entry{}, fmt.Errorf("check metadata ref: %w", err)
	}

	if !exists {
		return Entry{}, fmt.Errorf("%w: %s", errUnknownWorktreeID, worktreeID)
	}

	item, err := repo.Load(ctx, worktreeID)
	if err != nil {
		return Entry{}, fmt.Errorf("load metadata: %w", err)
	}

	return buildEntry(ctx, cwd, repo, item, withGitHub), nil
}

func buildEntry(ctx context.Context, cwd string, repo catalog.Repository, item catalog.Record, withGitHub bool) Entry {
	path := repo.WorktreePath(item.ID)
	branch, head, headOID := worktreeState(ctx, path)

	var prNumber, prState, prHeadOID string
	if withGitHub {
		prNumber, prState, prHeadOID = pullRequest(ctx, path, branch)
	}

	included := headIncluded(ctx, cwd, headOID)

	return Entry{
		ID:         item.ID,
		CreatedAt:  item.CreatedAt,
		Path:       path,
		Memo:       item.Memo,
		Branch:     branch,
		Head:       head,
		PRNumber:   prNumber,
		State:      resolveState(included, prState),
		Integrated: included || mergedPullRequestHead(prState, headOID, prHeadOID),
	}
}

// worktreeState reports the current branch (empty when detached) and the short
// and full HEAD commit hashes for the worktree at path. Errors are ignored: a
// worktree that is missing or otherwise unreadable simply reports empty values,
// since ls is a best-effort, read-only view.
func worktreeState(ctx context.Context, path string) (string, string, string) {
	runner := git.NewRunner(path)
	branch := ""
	head := ""
	headOID := ""

	if result, err := runner.Run(ctx, "symbolic-ref", "--short", "-q", "HEAD"); err == nil {
		branch = result.Stdout
	}

	if result, err := runner.Run(ctx, "rev-parse", "--short", "HEAD"); err == nil {
		head = result.Stdout
	}

	if result, err := runner.Run(ctx, "rev-parse", "HEAD"); err == nil {
		headOID = result.Stdout
	}

	return branch, head, headOID
}

func headIncluded(ctx context.Context, cwd string, head string) bool {
	if head == "" {
		return false
	}

	_, err := git.NewRunner(cwd).Run(ctx, "merge-base", "--is-ancestor", head, "HEAD")

	return err == nil
}

func resolveState(included bool, prState string) string {
	if included {
		return mergedState
	}

	return prState
}

type pullRequestView struct {
	Number     json.Number `json:"number"`
	State      string      `json:"state"`
	IsDraft    bool        `json:"isDraft"`
	HeadRefOID string      `json:"headRefOid"`
}

// pullRequest returns the number and state of the pull request associated with
// branch, resolved via the optional `gh` CLI. It returns empty values
// when gh is unavailable, the worktree is detached, or no PR is found, so
// ls remains usable without any GitHub integration configured.
func pullRequest(ctx context.Context, path string, branch string) (string, string, string) {
	if branch == "" {
		return "", "", ""
	}

	// gh interprets a leading "-" as a flag rather than a branch name; git
	// itself refuses to create such branches, but guard defensively since
	// branch is passed straight through as a positional argument below.
	if strings.HasPrefix(branch, "-") {
		return "", "", ""
	}

	if _, err := exec.LookPath(ghExecutable); err != nil {
		return "", "", ""
	}

	// #nosec G204 -- gh is invoked directly via exec (no shell), so branch cannot inject
	// additional shell commands. The only remaining risk is gh mistaking branch for a flag,
	// which the leading-"-" check above already rules out.
	cmd := exec.CommandContext(
		ctx,
		ghExecutable,
		"pr",
		"view",
		branch,
		"--json",
		"number,state,isDraft,headRefOid",
	)
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return "", "", ""
	}

	var view pullRequestView
	if err := json.Unmarshal(output, &view); err != nil {
		return "", "", ""
	}

	return view.Number.String(), pullRequestState(view), view.HeadRefOID
}

func mergedPullRequestHead(prState string, headOID string, prHeadOID string) bool {
	return prState == mergedState && headOID != "" && headOID == prHeadOID
}

func pullRequestState(view pullRequestView) string {
	if view.IsDraft {
		return "Draft"
	}

	switch view.State {
	case "OPEN":
		return "Open"
	case "MERGED":
		return mergedState
	case "CLOSED":
		return "Closed"
	default:
		return ""
	}
}
