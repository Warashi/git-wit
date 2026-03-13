// Package worktree manages git-wit worktree identifiers and paths.
package worktree

import (
	"fmt"
	"path/filepath"

	"github.com/oklog/ulid/v2"
)

const managedDirName = ".git-wit"

// NewID returns a new lexicographically sortable worktree identifier.
func NewID() string {
	return ulid.Make().String()
}

// ValidateID checks whether id is a valid ULID string.
func ValidateID(id string) error {
	_, err := ulid.ParseStrict(id)
	if err != nil {
		return fmt.Errorf("parse ulid: %w", err)
	}

	return nil
}

// RootDir returns the managed worktree root under a repository root.
func RootDir(repoRoot string) string {
	return filepath.Join(repoRoot, managedDirName, "worktrees")
}

// Path returns the absolute path for the managed worktree ID.
func Path(repoRoot string, id string) string {
	return filepath.Join(RootDir(repoRoot), id)
}
