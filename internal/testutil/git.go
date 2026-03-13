// Package testutil provides helpers for integration-style tests.
package testutil

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const testFilePerm = 0o600

// InitGitRepo creates a temporary git repository with one commit.
func InitGitRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	RunGit(t, repoDir, "init")
	RunGit(t, repoDir, "config", "user.name", "Test User")
	RunGit(t, repoDir, "config", "user.email", "test@example.com")

	filePath := filepath.Join(repoDir, "README.md")

	err := os.WriteFile(filePath, []byte("hello\n"), testFilePerm)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	RunGit(t, repoDir, "add", "README.md")
	RunGit(t, repoDir, "commit", "-m", "init")

	return repoDir
}

// RunGit executes git in the provided repository and fails the test on error.
func RunGit(t *testing.T, repoDir string, args ...string) []byte {
	t.Helper()

	return RunGitWithInput(t, repoDir, "", args...)
}

// RunGitWithInput executes git with stdin in the provided repository.
func RunGitWithInput(t *testing.T, repoDir string, stdin string, args ...string) []byte {
	t.Helper()

	// #nosec G204 -- tests invoke local git with controlled arguments.
	cmd := exec.CommandContext(context.Background(), "git", args...)

	cmd.Dir = repoDir

	cmd.Env = filteredGitEnv()
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}

	return output
}

func filteredGitEnv() []string {
	env := os.Environ()

	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, "GIT_COMMON_DIR=") {
			continue
		}

		if strings.HasPrefix(entry, "GIT_DIR=") {
			continue
		}

		if strings.HasPrefix(entry, "GIT_INDEX_FILE=") {
			continue
		}

		if strings.HasPrefix(entry, "GIT_WORK_TREE=") {
			continue
		}

		filtered = append(filtered, entry)
	}

	return filtered
}
