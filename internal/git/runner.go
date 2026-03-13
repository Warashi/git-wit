// Package git provides thin wrappers around the git executable.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const gitDirArgCount = 2

// Runner executes git commands relative to a repository path.
type Runner struct {
	repoDir string
}

// Result stores stdout and stderr from a git invocation.
type Result struct {
	Stdout string
	Stderr string
}

// CommandError wraps a failing git subprocess.
type CommandError struct {
	args []string
	err  error
}

// NewRunner creates a git command runner.
func NewRunner(repoDir string) Runner {
	return Runner{repoDir: repoDir}
}

// Run executes a git command and captures trimmed stdout and stderr.
func (r Runner) Run(ctx context.Context, args ...string) (Result, error) {
	return r.run(ctx, "", args...)
}

// RunInput executes a git command with stdin content.
func (r Runner) RunInput(ctx context.Context, stdin string, args ...string) (Result, error) {
	return r.run(ctx, stdin, args...)
}

func (r Runner) run(ctx context.Context, stdin string, args ...string) (Result, error) {
	cmdArgs := make([]string, 0, len(args)+gitDirArgCount)
	if r.repoDir != "" {
		cmdArgs = append(cmdArgs, "-C", r.repoDir)
	}

	cmdArgs = append(cmdArgs, args...)

	// #nosec G204 -- git is the intended executable and arguments come from trusted callers.
	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	cmd.Env = filteredGitEnv()

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := Result{
		Stdout: strings.TrimRight(stdout.String(), "\n"),
		Stderr: strings.TrimRight(stderr.String(), "\n"),
	}
	if err != nil {
		return result, CommandError{
			args: cmdArgs,
			err:  fmt.Errorf("%w: %s", err, result.Stderr),
		}
	}

	return result, nil
}

func (e CommandError) Error() string {
	return "git " + strings.Join(e.args, " ") + ": " + e.err.Error()
}

func (e CommandError) Unwrap() error {
	return e.err
}

// ExitCode returns the subprocess exit code when available.
func (e CommandError) ExitCode() int {
	var exitErr *exec.ExitError
	if !errors.As(e.err, &exitErr) {
		return 0
	}

	return exitErr.ExitCode()
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
