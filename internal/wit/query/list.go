package query

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Warashi/git-wit/internal/git"
	"github.com/Warashi/git-wit/internal/wit/catalog"
	"golang.org/x/sync/errgroup"
)

const (
	ghExecutable = "gh"
	mergedState  = "Merged"

	// minListParallelism is a floor on top of GOMAXPROCS: buildEntry's git/gh
	// subprocess calls are I/O-bound, so constrained environments (e.g. a
	// single-vCPU CI container) still benefit from more concurrency than CPU
	// count alone would allow.
	minListParallelism = 8

	// minPullRequestListLimit and pullRequestListLimitPerBranch size the
	// `gh pr list` batch window: large enough for small worktree counts to
	// comfortably cover recent activity, and scaling up for larger ones so
	// the single call still has a realistic chance of covering every branch.
	minPullRequestListLimit       = 50
	pullRequestListLimitPerBranch = 4
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

	limit := max(runtime.GOMAXPROCS(0), minListParallelism)

	states := make([]worktreeInfo, len(items))

	var localGroup errgroup.Group

	localGroup.SetLimit(limit)

	for i, item := range items {
		localGroup.Go(func() error {
			states[i].path = repo.WorktreePath(item.ID)
			states[i].branch, states[i].head, states[i].headOID = worktreeState(ctx, states[i].path)

			return nil
		})
	}

	_ = localGroup.Wait() // worktreeState is best-effort and never returns an error.

	var prByBranch map[string]pullRequestView

	if withGitHub {
		branches := make([]string, len(states))
		for i, state := range states {
			branches[i] = state.branch
		}

		prByBranch = fetchPullRequests(ctx, repo.Root(), branches, limit)
	}

	entries := make([]Entry, len(items))

	var entryGroup errgroup.Group

	entryGroup.SetLimit(limit)

	for i, item := range items {
		localState := states[i]

		entryGroup.Go(func() error {
			included := headIncluded(ctx, cwd, localState.headOID)
			entries[i] = newEntry(item, localState, prByBranch[localState.branch], included, withGitHub)

			return nil
		})
	}

	_ = entryGroup.Wait() // headIncluded is best-effort and never returns an error.

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

	var state worktreeInfo

	state.path = repo.WorktreePath(item.ID)
	state.branch, state.head, state.headOID = worktreeState(ctx, state.path)

	var view pullRequestView

	if withGitHub {
		view = viewPullRequest(ctx, state.path, state.branch)
	}

	included := headIncluded(ctx, cwd, state.headOID)

	return newEntry(item, state, view, included, withGitHub), nil
}

// worktreeInfo is the local (gh-independent) state of one managed worktree.
type worktreeInfo struct {
	path    string
	branch  string
	head    string
	headOID string
}

// newEntry assembles the read model for one worktree from its local state
// and (if resolved) pull request info. A zero-value view is indistinguishable
// from "no pull request found", which keeps this a pure combination step.
// State stays empty unless withGitHub is set: the CLI contract promises a
// null state without --full, while Integrated always carries the local
// ancestor judgement.
func newEntry(
	item catalog.Record,
	state worktreeInfo,
	view pullRequestView,
	included bool,
	withGitHub bool,
) Entry {
	prState := pullRequestState(view)

	resolvedState := ""
	if withGitHub {
		resolvedState = resolveState(included, prState)
	}

	return Entry{
		ID:         item.ID,
		CreatedAt:  item.CreatedAt,
		Path:       state.path,
		Memo:       item.Memo,
		Branch:     state.branch,
		Head:       state.head,
		PRNumber:   view.Number.String(),
		State:      resolvedState,
		Integrated: included || mergedPullRequestHead(prState, state.headOID, view.HeadRefOID),
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

// viewPullRequest resolves the pull request associated with branch via the
// optional `gh` CLI, run from dir. It returns a zero value when gh is
// unavailable, the worktree is detached, or no PR is found, so ls remains
// usable without any GitHub integration configured.
func viewPullRequest(ctx context.Context, dir string, branch string) pullRequestView {
	var notFound pullRequestView

	if branch == "" {
		return notFound
	}

	// gh interprets a leading "-" as a flag rather than a branch name; git
	// itself refuses to create such branches, but guard defensively since
	// branch is passed straight through as a positional argument below.
	if strings.HasPrefix(branch, "-") {
		return notFound
	}

	// gh pr view treats an all-digit argument as a PR number, so a branch
	// named e.g. "1234" would resolve an unrelated pull request; skip the
	// lookup rather than report wrong data.
	if isAllDigits(branch) {
		return notFound
	}

	if _, err := exec.LookPath(ghExecutable); err != nil {
		return notFound
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
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		return notFound
	}

	var view pullRequestView
	if err := json.Unmarshal(output, &view); err != nil {
		return notFound
	}

	return view
}

// pullRequestListView is one row of `gh pr list --json ...` output: the same
// shape as pullRequestView, plus the branch name needed to match it back to
// a worktree.
type pullRequestListView struct {
	Number      json.Number `json:"number"`
	HeadRefName string      `json:"headRefName"`
	State       string      `json:"state"`
	IsDraft     bool        `json:"isDraft"`
	HeadRefOID  string      `json:"headRefOid"`
}

// fetchPullRequests resolves pull request info for branches (blank entries
// are skipped) with a single `gh pr list` call, matched by branch name.
// Branches missing from that result are looked up individually and in
// parallel as a fallback, since gh pr list's window can miss older pull
// requests. If gh is unavailable or the batch call itself fails, no branch
// gets PR info: a failure there is likely environmental (offline, rate
// limited) and would just as likely make N individual fallback calls fail
// too, so ls stays best-effort rather than retrying expensively.
func fetchPullRequests(
	ctx context.Context,
	repoRoot string,
	branches []string,
	parallelism int,
) map[string]pullRequestView {
	distinct := distinctNonEmptyBranches(branches)
	if len(distinct) == 0 {
		return nil
	}

	if _, err := exec.LookPath(ghExecutable); err != nil {
		return nil
	}

	listed, ok := listPullRequests(ctx, repoRoot, len(distinct))
	if !ok {
		return nil
	}

	missing := make([]string, 0, len(distinct))

	for _, branch := range distinct {
		if _, found := listed[branch]; !found {
			missing = append(missing, branch)
		}
	}

	if len(missing) == 0 {
		return listed
	}

	var listedMu sync.Mutex

	var group errgroup.Group

	group.SetLimit(parallelism)

	for _, branch := range missing {
		group.Go(func() error {
			view := viewPullRequest(ctx, repoRoot, branch)

			listedMu.Lock()
			listed[branch] = view
			listedMu.Unlock()

			return nil
		})
	}

	_ = group.Wait() // viewPullRequest is best-effort and never returns an error.

	return listed
}

// listPullRequests fetches a branch-count-scaled window of pull requests in
// one gh call and indexes them by branch name. ok is false only when the
// call itself fails (gh unavailable, network error, rate limit, ...); a
// successful call with no matches returns a non-nil, empty map.
func listPullRequests(
	ctx context.Context,
	repoRoot string,
	distinctBranchCount int,
) (map[string]pullRequestView, bool) {
	limit := max(minPullRequestListLimit, distinctBranchCount*pullRequestListLimitPerBranch)

	// #nosec G204 -- every argument is a fixed literal; no external input reaches argv here.
	cmd := exec.CommandContext(
		ctx,
		ghExecutable,
		"pr",
		"list",
		"--state", "all",
		"--limit", strconv.Itoa(limit),
		"--json", "number,headRefName,state,isDraft,headRefOid",
	)
	cmd.Dir = repoRoot

	output, err := cmd.Output()
	if err != nil {
		return nil, false
	}

	var listed []pullRequestListView
	if err := json.Unmarshal(output, &listed); err != nil {
		return nil, false
	}

	result := make(map[string]pullRequestView, len(listed))
	for _, view := range listed {
		// gh pr list returns pull requests newest-first; keep the first
		// match so a branch reused for a new PR is not reported with a
		// stale, older one.
		if _, exists := result[view.HeadRefName]; exists {
			continue
		}

		result[view.HeadRefName] = pullRequestView{
			Number:     view.Number,
			State:      view.State,
			IsDraft:    view.IsDraft,
			HeadRefOID: view.HeadRefOID,
		}
	}

	return result, true
}

func isAllDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}

	return value != ""
}

func distinctNonEmptyBranches(branches []string) []string {
	seen := make(map[string]bool, len(branches))
	distinct := make([]string, 0, len(branches))

	for _, branch := range branches {
		if branch == "" || seen[branch] {
			continue
		}

		seen[branch] = true

		distinct = append(distinct, branch)
	}

	return distinct
}

func mergedPullRequestHead(prState string, headOID string, prHeadOID string) bool {
	return prState == mergedState && headOID != "" && headOID == prHeadOID
}

func pullRequestState(view pullRequestView) string {
	switch view.State {
	case "OPEN":
		// Draft only qualifies an open pull request; GitHub keeps
		// isDraft set on PRs that were closed while still drafts.
		if view.IsDraft {
			return "Draft"
		}

		return "Open"
	case "MERGED":
		return mergedState
	case "CLOSED":
		return "Closed"
	default:
		return ""
	}
}
