package query_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
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

	entries, err := query.List(context.Background(), repoDir)
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

	entries, err := query.List(context.Background(), repoDir)
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

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "open", output: `{"number":42,"state":"OPEN","isDraft":false}`, want: "Open"},
		{name: "draft", output: `{"number":42,"state":"OPEN","isDraft":true}`, want: "Draft"},
		{name: "merged", output: `{"number":42,"state":"MERGED","isDraft":false}`, want: "Merged"},
		{name: "closed", output: `{"number":42,"state":"CLOSED","isDraft":false}`, want: "Closed"},
	}

	for _, test := range tests {
		writeFakeGH(t, ghPath, test.output)

		entries, err := query.List(context.Background(), repoDir)
		if err != nil {
			t.Fatalf("%s: List() error = %v", test.name, err)
		}

		if got := entries[0].PRNumber; got != "42" {
			t.Fatalf("%s: entry.PRNumber = %q, want %q", test.name, got, "42")
		}

		if got := entries[0].State; got != test.want {
			t.Fatalf("%s: entry.State = %q, want %q", test.name, got, test.want)
		}
	}

	writeFakeGH(t, ghPath, `{"number":42,"state":"CLOSED","isDraft":false}`)
	testutil.RunGit(t, repoDir, "merge", "--ff-only", "feature/state")

	entries, err := query.List(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("List() after merge error = %v", err)
	}

	if got := entries[0].State; got != mergedState {
		t.Fatalf("entry.State after merge = %q, want %q", got, mergedState)
	}
}

func writeFakeGH(t *testing.T, path string, output string) {
	t.Helper()

	content := "#!/bin/sh\nprintf '%s\\n' '" + output + "'\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	if err := os.Chmod(path, 0o700); err != nil { // #nosec G302 -- the fake gh must be executable.
		t.Fatalf("Chmod(%q) error = %v", path, err)
	}
}
