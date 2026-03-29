package sync_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Warashi/git-wit/internal/wit/sync"
)

func TestRunExecutesCommandsInWorkingDirectory(t *testing.T) {
	t.Setenv("SHELL", "")

	cwd := t.TempDir()

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	err := sync.Run(context.Background(), cwd, []string{`pwd > seen.txt`}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// #nosec G304 -- test reads from the temporary directory it created.
	got, err := os.ReadFile(filepath.Join(cwd, "seen.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if strings.TrimSpace(string(got)) != cwd {
		t.Fatalf("cwd = %q, want %q", strings.TrimSpace(string(got)), cwd)
	}
}

func TestRunStopsOnFirstFailure(t *testing.T) {
	t.Setenv("SHELL", "")

	cwd := t.TempDir()

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	err := sync.Run(context.Background(), cwd, []string{
		`printf first > order.txt`,
		`false`,
		`printf third > order.txt`,
	}, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}

	// #nosec G304 -- test reads from the temporary directory it created.
	got, err := os.ReadFile(filepath.Join(cwd, "order.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(got) != "first" {
		t.Fatalf("order.txt = %q, want %q", string(got), "first")
	}
}
