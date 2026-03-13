package git_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Warashi/git-wit/internal/git"
)

func TestRunnerRun(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepo(t)
	runner := git.NewRunner(repoDir)

	result, err := runner.Run(context.Background(), "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Stdout != repoDir {
		t.Fatalf("Run() stdout = %q, want %q", result.Stdout, repoDir)
	}
}

func TestRunnerRun_CommandError(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepo(t)
	runner := git.NewRunner(repoDir)

	_, err := runner.Run(context.Background(), "show", "refs/does-not-exist")
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}

	var cmdErr git.CommandError
	if !errors.As(err, &cmdErr) {
		t.Fatalf("Run() error type = %T, want CommandError", err)
	}

	if cmdErr.ExitCode() == 0 {
		t.Fatalf("ExitCode() = %d, want non-zero", cmdErr.ExitCode())
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runCommand(t, repoDir, "init")
	runCommand(t, repoDir, "config", "user.name", "Test User")
	runCommand(t, repoDir, "config", "user.email", "test@example.com")

	filePath := filepath.Join(repoDir, "README.md")

	err := os.WriteFile(filePath, []byte("hello\n"), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	runCommand(t, repoDir, "add", "README.md")
	runCommand(t, repoDir, "commit", "-m", "init")

	return repoDir
}

func runCommand(t *testing.T, dir string, args ...string) {
	t.Helper()

	// #nosec G204 -- tests invoke local git with controlled arguments.
	cmd := exec.CommandContext(context.Background(), "git", args...)

	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
