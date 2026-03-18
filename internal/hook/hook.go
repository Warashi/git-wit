// Package hook executes git-wit hook command strings.
package hook

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

var errNoShellExecutable = errors.New("no shell executable found")

// Run executes shell command strings sequentially in cwd.
func Run(ctx context.Context, cwd string, commands []string, stdout io.Writer, stderr io.Writer) error {
	if stdout == nil {
		stdout = io.Discard
	}

	if stderr == nil {
		stderr = io.Discard
	}

	shell, err := resolveShell()
	if err != nil {
		return fmt.Errorf("resolve shell: %w", err)
	}

	for index, command := range commands {
		if strings.TrimSpace(command) == "" {
			continue
		}

		err = runCommand(ctx, shell, cwd, command, stdout, stderr)
		if err != nil {
			return fmt.Errorf("run hook %d: %w", index+1, err)
		}
	}

	return nil
}

func runCommand(
	ctx context.Context,
	shell string,
	cwd string,
	command string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	// #nosec G204 -- hook command strings are intentional shell input from git config.
	cmd := exec.CommandContext(ctx, shell, "-c", command)
	cmd.Dir = cwd
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("execute %q: %w", command, err)
	}

	return nil
}

func resolveShell() (string, error) {
	candidates := []string{os.Getenv("SHELL"), "bash", "sh"}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		path, err := exec.LookPath(candidate)
		if err == nil {
			return path, nil
		}
	}

	return "", errNoShellExecutable
}
