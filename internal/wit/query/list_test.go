package query_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Warashi/git-wit/internal/testutil"
	"github.com/Warashi/git-wit/internal/wit/create"
	"github.com/Warashi/git-wit/internal/wit/query"
)

var shortHashPattern = regexp.MustCompile(`^[0-9a-f]{4,40}$`)

const mergedState = "Merged"

func TestListReportsDetachedHeadWorktree(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	created, err := create.Create(context.Background(), repoDir, time.Unix(100, 0), "memo", nil, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	entries, err := query.List(context.Background(), repoDir, true)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("List() = %d entries, want 1", len(entries))
	}

	entry := entries[0]
	if entry.ID != created.ID {
		t.Fatalf("entry.ID = %q, want %q", entry.ID, created.ID)
	}

	if entry.Branch != "" {
		t.Fatalf("entry.Branch = %q, want empty (detached HEAD)", entry.Branch)
	}

	if !shortHashPattern.MatchString(entry.Head) {
		t.Fatalf("entry.Head = %q, want a short commit hash", entry.Head)
	}

	if entry.PRNumber != "" {
		t.Fatalf("entry.PRNumber = %q, want empty", entry.PRNumber)
	}

	if entry.State != mergedState {
		t.Fatalf("entry.State = %q, want %q", entry.State, mergedState)
	}

	if !entry.Integrated {
		t.Fatal("entry.Integrated = false, want true")
	}
}

func TestListReportsCheckedOutBranch(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	created, err := create.Create(context.Background(), repoDir, time.Unix(200, 0), "memo", nil, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	testutil.RunGit(t, created.Path, "checkout", "-b", "feature/example")

	entries, err := query.List(context.Background(), repoDir, true)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("List() = %d entries, want 1", len(entries))
	}

	if got := entries[0].Branch; got != "feature/example" {
		t.Fatalf("entry.Branch = %q, want %q", got, "feature/example")
	}

	if !shortHashPattern.MatchString(entries[0].Head) {
		t.Fatalf("entry.Head = %q, want a short commit hash", entries[0].Head)
	}

	if got := entries[0].State; got != mergedState {
		t.Fatalf("entry.State = %q, want %q", got, mergedState)
	}
}

func TestListReportsPullRequestStateUntilHeadIsIncluded(t *testing.T) {
	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	created, err := create.Create(context.Background(), repoDir, time.Unix(300, 0), "memo", nil, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	testutil.RunGit(t, created.Path, "checkout", "-b", "feature/state")
	testutil.RunGit(t, created.Path, "commit", "--allow-empty", "-m", "feature")
	headOID := strings.TrimSpace(string(testutil.RunGit(t, created.Path, "rev-parse", "HEAD")))
	mismatchedOID := strings.Repeat("0", 40)

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	tests := []pullRequestStateTest{
		{name: "open", output: `{"number":42,"state":"OPEN","isDraft":false}`, want: "Open", wantIntegrated: false},
		{name: "draft", output: `{"number":42,"state":"OPEN","isDraft":true}`, want: "Draft", wantIntegrated: false},
		{
			name:           "merged at local head",
			output:         fmt.Sprintf(`{"number":42,"state":"MERGED","isDraft":false,"headRefOid":%q}`, headOID),
			want:           "Merged",
			wantIntegrated: true,
		},
		{
			name:           "merged before local head",
			output:         fmt.Sprintf(`{"number":42,"state":"MERGED","isDraft":false,"headRefOid":%q}`, mismatchedOID),
			want:           "Merged",
			wantIntegrated: false,
		},
		{name: "closed", output: `{"number":42,"state":"CLOSED","isDraft":false}`, want: "Closed", wantIntegrated: false},
	}

	for _, test := range tests {
		assertPullRequestState(t, repoDir, ghPath, "feature/state", test)
	}

	writeFakeGH(t, ghPath, "feature/state", `{"number":42,"state":"CLOSED","isDraft":false}`)
	testutil.RunGit(t, repoDir, "merge", "--ff-only", "feature/state")

	entries, err := query.List(context.Background(), repoDir, true)
	if err != nil {
		t.Fatalf("List() after merge error = %v", err)
	}

	if got := entries[0].State; got != mergedState {
		t.Fatalf("entry.State after merge = %q, want %q", got, mergedState)
	}

	if !entries[0].Integrated {
		t.Fatal("entry.Integrated after merge = false, want true")
	}
}

func TestListFallsBackToIndividualLookupWhenBatchListMisses(t *testing.T) {
	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	created, err := create.Create(context.Background(), repoDir, time.Unix(400, 0), "memo", nil, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	testutil.RunGit(t, created.Path, "checkout", "-b", "feature/missing-from-batch")
	testutil.RunGit(t, created.Path, "commit", "--allow-empty", "-m", "feature")

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// The batch call succeeds but returns no matches (e.g. an older PR
	// outside its window); List must fall back to an individual lookup.
	writeFakeGHList(t, ghPath, "0", "[]", `{"number":7,"state":"OPEN","isDraft":false}`)

	entries, err := query.List(context.Background(), repoDir, true)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if got := entries[0].PRNumber; got != "7" {
		t.Fatalf("entry.PRNumber = %q, want %q (from individual fallback)", got, "7")
	}

	if got := entries[0].State; got != "Open" {
		t.Fatalf("entry.State = %q, want %q", got, "Open")
	}
}

func TestListSkipsFallbackWhenBatchListFails(t *testing.T) {
	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)

	created, err := create.Create(context.Background(), repoDir, time.Unix(500, 0), "memo", nil, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	testutil.RunGit(t, created.Path, "checkout", "-b", "feature/batch-fails")

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// gh pr view would succeed if called, but an outright batch failure
	// (network error, rate limit, ...) must not trigger a per-branch
	// fallback for every worktree.
	writeFakeGHList(t, ghPath, "1", "", `{"number":9,"state":"OPEN","isDraft":false}`)

	entries, err := query.List(context.Background(), repoDir, true)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if got := entries[0].PRNumber; got != "" {
		t.Fatalf("entry.PRNumber = %q, want empty (no fallback after batch failure)", got)
	}
}

type pullRequestStateTest struct {
	name           string
	output         string
	want           string
	wantIntegrated bool
}

func assertPullRequestState(t *testing.T, repoDir string, ghPath string, branch string, test pullRequestStateTest) {
	t.Helper()

	writeFakeGH(t, ghPath, branch, test.output)

	entries, err := query.List(context.Background(), repoDir, true)
	if err != nil {
		t.Fatalf("%s: List() error = %v", test.name, err)
	}

	if got := entries[0].PRNumber; got != "42" {
		t.Fatalf("%s: entry.PRNumber = %q, want %q", test.name, got, "42")
	}

	if got := entries[0].State; got != test.want {
		t.Fatalf("%s: entry.State = %q, want %q", test.name, got, test.want)
	}

	if got := entries[0].Integrated; got != test.wantIntegrated {
		t.Fatalf(
			"%s: entry.Integrated = %t, want %t",
			test.name,
			got,
			test.wantIntegrated,
		)
	}
}

// writeFakeGH writes a fake `gh` that responds to `gh pr list` with a
// single-element array (viewOutput plus a matching headRefName) and to
// `gh pr view` with viewOutput as-is, so List's batch lookup finds the pull
// request on its first call without needing the individual fallback.
func writeFakeGH(t *testing.T, path string, branch string, viewOutput string) {
	t.Helper()

	listOutput := "[" + withHeadRefName(viewOutput, branch) + "]"

	content := "#!/bin/sh\n" +
		"if [ \"$1\" = pr ] && [ \"$2\" = list ]; then\n" +
		"  printf '%s\\n' '" + listOutput + "'\n" +
		"else\n" +
		"  printf '%s\\n' '" + viewOutput + "'\n" +
		"fi\n"

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	if err := os.Chmod(path, 0o700); err != nil { // #nosec G302 -- the fake gh must be executable.
		t.Fatalf("Chmod(%q) error = %v", path, err)
	}
}

func withHeadRefName(object string, branch string) string {
	return strings.TrimSuffix(object, "}") + fmt.Sprintf(",%q:%q}", "headRefName", branch)
}

// writeFakeGHList writes a fake `gh` where `gh pr list` prints listOutput
// verbatim (e.g. to simulate a miss or an outright failure) and `gh pr view`
// prints viewOutput, so tests can exercise List's fallback and
// batch-failure paths independently of the happy path above.
func writeFakeGHList(t *testing.T, path string, listExit string, listOutput string, viewOutput string) {
	t.Helper()

	content := "#!/bin/sh\n" +
		"if [ \"$1\" = pr ] && [ \"$2\" = list ]; then\n" +
		"  printf '%s\\n' '" + listOutput + "'\n" +
		"  exit " + listExit + "\n" +
		"else\n" +
		"  printf '%s\\n' '" + viewOutput + "'\n" +
		"fi\n"

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	if err := os.Chmod(path, 0o700); err != nil { // #nosec G302 -- the fake gh must be executable.
		t.Fatalf("Chmod(%q) error = %v", path, err)
	}
}
