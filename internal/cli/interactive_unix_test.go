//go:build unix

package cli_test

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Warashi/git-wit/internal/testutil"
)

// /dev/null is a character device but not a terminal: confirmation-guarded
// commands must demand --yes instead of reading EOF and quietly doing
// nothing.
func TestRootCommandRemoveMergedRequiresYesForNullDeviceInput(t *testing.T) {
	t.Parallel()

	repoDir := testutil.InitGitRepo(t)
	configureWorktreeRoot(t, repoDir)
	created := createManagedWorktree(t, repoDir, time.Unix(100, 0))
	integrateManagedWorktree(t, repoDir, created)

	input, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", os.DevNull, err)
	}

	t.Cleanup(func() {
		if closeErr := input.Close(); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})

	var stdout bytes.Buffer

	cmd := newTestRootCommand(repoDir, time.Unix(200, 0))
	cmd.SetIn(input)
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"rm", "--merged"})

	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "requires --yes") {
		t.Fatalf("Execute() error = %v, want requires --yes", err)
	}

	assertManagedMetadata(t, repoDir, created.ID, true)
}
