package cli_test

import (
	"strings"
	"testing"
	"time"
)

func TestRootCommandPruneRejectsYesWithoutSystem(t *testing.T) {
	t.Parallel()

	cmd := newTestRootCommand(t.TempDir(), time.Unix(100, 0))
	cmd.SetArgs([]string{"prune", "--yes"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--yes requires --system") {
		t.Fatalf("Execute() error = %v, want --yes requires --system", err)
	}
}
