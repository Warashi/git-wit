package main

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestModuleRootBuildsAndShowsHelp(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "git-wit")
	ctx := context.Background()

	//nolint:gosec // The test builds the repository root into a temp file under our control.
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".")

	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build error = %v, output = %s", err, buildOutput)
	}

	//nolint:gosec // The test executes the binary it just built in a temp directory.
	helpCmd := exec.CommandContext(ctx, binaryPath, "--help")

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	helpCmd.Stdout = &stdout
	helpCmd.Stderr = &stderr

	err = helpCmd.Run()
	if err != nil {
		t.Fatalf("help command error = %v, stdout = %q, stderr = %q", err, stdout.String(), stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Usage:\n  git-wit [command]") {
		t.Fatalf("help stdout = %q", output)
	}
}
