package cli_test

import (
	"testing"

	"github.com/Warashi/git-wit/internal/cli"
)

func TestNewRootCommand_HasSubcommands(t *testing.T) {
	t.Parallel()

	cmd := cli.NewRootCommand()

	want := []string{"add", "ls", "dir", "rm", "merge", "prune"}
	for _, name := range want {
		got, _, err := cmd.Find([]string{name})
		if err != nil || got == cmd {
			t.Fatalf("Find(%q) error = %v, got root = %t", name, err, got == cmd)
		}
	}
}
