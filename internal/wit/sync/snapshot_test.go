package sync_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Warashi/git-wit/internal/git"
	"github.com/Warashi/git-wit/internal/testutil"
	"github.com/Warashi/git-wit/internal/wit/sync"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	cfg := sync.Config{
		IgnoredDefault:   sync.ModeNone,
		UntrackedDefault: sync.ModeNone,
		NoSyncPaths:      []string{"tmp/*", "tmp/**"},
		SymlinkPaths:     nil,
		CopyPaths:        []string{"**/node_modules", ".env.local"},
		AddHooks:         nil,
		WorktreeRoot:     "",
	}

	cases := []struct {
		name string
		path string
		kind sync.Kind
		want sync.Mode
	}{
		{name: "nosync wins", path: "tmp/app.log", kind: sync.KindUntracked, want: sync.ModeNone},
		{name: "nosync recursive glob", path: "tmp/packages/app.log", kind: sync.KindUntracked, want: sync.ModeNone},
		{name: "copy recursive root", path: "node_modules/", kind: sync.KindIgnored, want: sync.ModeCopy},
		{
			name: "copy recursive nested",
			path: "packages/web/node_modules/",
			kind: sync.KindIgnored,
			want: sync.ModeCopy,
		},
		{name: "copy explicit", path: ".env.local", kind: sync.KindUntracked, want: sync.ModeCopy},
		{name: "ignored default", path: "dist/app.js", kind: sync.KindIgnored, want: sync.ModeNone},
		{name: "untracked default", path: "scratch.txt", kind: sync.KindUntracked, want: sync.ModeNone},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := sync.Resolve(cfg, testCase.path, testCase.kind)
			if got != testCase.want {
				t.Fatalf("Resolve() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestApplyCopiesRecursiveGlobPath(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	destDir := t.TempDir()
	writeFile(t, filepath.Join(repoDir, ".gitignore"), "**/node_modules/\n")
	testutil.RunGit(t, repoDir, "add", ".gitignore")
	testutil.RunGit(t, repoDir, "commit", "-m", "add ignore rules")

	mustMkdir(t, filepath.Join(repoDir, "packages", "web", "node_modules"))
	writeFile(t, filepath.Join(repoDir, "packages", "web", "node_modules", "pkg.json"), "{}")

	//nolint:exhaustruct // Test input only sets fields relevant to resolution.
	cfg := sync.Config{
		IgnoredDefault:   sync.ModeNone,
		UntrackedDefault: sync.ModeNone,
		CopyPaths:        []string{"**/node_modules"},
		AddHooks:         nil,
		WorktreeRoot:     "",
	}

	err := sync.Apply(context.Background(), git.NewRunner(repoDir), repoDir, destDir, cfg)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// #nosec G304 -- test reads from the temporary destination directory it created.
	got, err := os.ReadFile(filepath.Join(destDir, "packages", "web", "node_modules", "pkg.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(got) != "{}" {
		t.Fatalf("copied pkg = %q, want %q", string(got), "{}")
	}
}

func TestCollect(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)

	writeFile(t, filepath.Join(repoDir, ".gitignore"), "ignored/\n*.log\n")
	testutil.RunGit(t, repoDir, "add", ".gitignore")
	testutil.RunGit(t, repoDir, "commit", "-m", "add ignore rules")

	mustMkdir(t, filepath.Join(repoDir, "ignored"))
	writeFile(t, filepath.Join(repoDir, "ignored", "a.txt"), "ignored")
	writeFile(t, filepath.Join(repoDir, "scratch.txt"), "scratch")

	items, err := sync.Collect(context.Background(), git.NewRunner(repoDir))
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("len(Collect()) = %d, want 2", len(items))
	}

	if items[0].Path != "ignored/" || items[0].Kind != sync.KindIgnored {
		t.Fatalf("Collect()[0] = %#v", items[0])
	}

	if items[1].Path != "scratch.txt" || items[1].Kind != sync.KindUntracked {
		t.Fatalf("Collect()[1] = %#v", items[1])
	}
}

//nolint:gosmopolitan // Non-ASCII path handling is exactly what these tests cover.
const (
	nonASCIIIgnoredDir    = "無視ディレクトリ"
	nonASCIIIgnoredFile   = "中身.txt"
	nonASCIIUntrackedFile = "日本語 メモ.txt"
)

// setupNonASCIIRepo creates a repository with an ignored directory and an
// untracked file whose names git C-quotes under the default core.quotePath.
func setupNonASCIIRepo(t *testing.T) string {
	t.Helper()

	repoDir := testutil.InitGitRepo(t)

	writeFile(t, filepath.Join(repoDir, ".gitignore"), nonASCIIIgnoredDir+"/\n")
	testutil.RunGit(t, repoDir, "add", ".gitignore")
	testutil.RunGit(t, repoDir, "commit", "-m", "add ignore rules")

	mustMkdir(t, filepath.Join(repoDir, nonASCIIIgnoredDir))
	writeFile(t, filepath.Join(repoDir, nonASCIIIgnoredDir, nonASCIIIgnoredFile), "ignored")
	writeFile(t, filepath.Join(repoDir, nonASCIIUntrackedFile), "untracked")

	return repoDir
}

func TestCollectHandlesNonASCIIPaths(t *testing.T) {
	t.Parallel()

	repoDir := setupNonASCIIRepo(t)

	items, err := sync.Collect(context.Background(), git.NewRunner(repoDir))
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("len(Collect()) = %d, want 2: %#v", len(items), items)
	}

	if items[0].Path != nonASCIIUntrackedFile || items[0].Kind != sync.KindUntracked {
		t.Fatalf("Collect()[0] = %#v", items[0])
	}

	if items[1].Path != nonASCIIIgnoredDir+"/" || items[1].Kind != sync.KindIgnored {
		t.Fatalf("Collect()[1] = %#v", items[1])
	}
}

func TestApplyCopiesNonASCIIPaths(t *testing.T) {
	t.Parallel()

	repoDir := setupNonASCIIRepo(t)
	destDir := t.TempDir()

	//nolint:exhaustruct // Test input only sets fields relevant to resolution.
	cfg := sync.Config{
		IgnoredDefault:   sync.ModeCopy,
		UntrackedDefault: sync.ModeCopy,
	}

	err := sync.Apply(context.Background(), git.NewRunner(repoDir), repoDir, destDir, cfg)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	assertFileContents(t, filepath.Join(destDir, nonASCIIUntrackedFile), "untracked")
	assertFileContents(t, filepath.Join(destDir, nonASCIIIgnoredDir, nonASCIIIgnoredFile), "ignored")
}

func assertFileContents(t *testing.T, filePath string, want string) {
	t.Helper()

	// #nosec G304 -- test reads from the temporary destination directory it created.
	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", filePath, err)
	}

	if string(got) != want {
		t.Fatalf("contents of %q = %q, want %q", filePath, string(got), want)
	}
}

func TestApply(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	destDir := t.TempDir()
	writeFile(t, filepath.Join(repoDir, ".gitignore"), "node_modules/\n")
	testutil.RunGit(t, repoDir, "add", ".gitignore")
	testutil.RunGit(t, repoDir, "commit", "-m", "add ignore rules")

	mustMkdir(t, filepath.Join(repoDir, "node_modules"))
	writeFile(t, filepath.Join(repoDir, "node_modules", "pkg.json"), "{}")
	writeFile(t, filepath.Join(repoDir, ".env.local"), "A=1\n")
	writeFile(t, filepath.Join(repoDir, "scratch.txt"), "skip\n")

	//nolint:exhaustruct // Test input only sets fields relevant to resolution.
	cfg := sync.Config{
		IgnoredDefault:   sync.ModeNone,
		UntrackedDefault: sync.ModeNone,
		SymlinkPaths:     []string{"node_modules"},
		CopyPaths:        []string{".env.local"},
		AddHooks:         nil,
		WorktreeRoot:     "",
	}

	err := sync.Apply(context.Background(), git.NewRunner(repoDir), repoDir, destDir, cfg)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// #nosec G304 -- test reads from the temporary destination directory it created.
	envBytes, err := os.ReadFile(filepath.Join(destDir, ".env.local"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(envBytes) != "A=1\n" {
		t.Fatalf("copied env = %q", string(envBytes))
	}

	linkTarget, err := os.Readlink(filepath.Join(destDir, "node_modules"))
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}

	wantTarget := filepath.Join(repoDir, "node_modules")
	if linkTarget != wantTarget {
		t.Fatalf("Readlink() = %q, want %q", linkTarget, wantTarget)
	}

	_, err = os.Stat(filepath.Join(destDir, "scratch.txt"))
	if !os.IsNotExist(err) {
		t.Fatalf("scratch.txt stat error = %v, want not exist", err)
	}
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()

	err := os.MkdirAll(dir, 0o750)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
}

func writeFile(t *testing.T, filePath string, contents string) {
	t.Helper()

	err := os.WriteFile(filePath, []byte(contents), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
