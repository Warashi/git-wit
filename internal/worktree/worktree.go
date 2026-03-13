// Package worktree manages git-wit worktree identifiers and paths.
package worktree

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
)

const uuidVersion7 = 7

var (
	errInvalidUUIDVersion = errors.New("invalid uuid version")
	errNonCanonicalUUIDv7 = errors.New("non-canonical uuidv7")
)

// NewID returns a new lexicographically sortable worktree identifier.
func NewID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// ValidateID checks whether worktreeID is a canonical UUIDv7 string.
func ValidateID(worktreeID string) error {
	parsed, err := uuid.Parse(worktreeID)
	if err != nil {
		return fmt.Errorf("parse uuid: %w", err)
	}

	if parsed.Version() != uuidVersion7 {
		return fmt.Errorf("%w: got %d, want %d", errInvalidUUIDVersion, parsed.Version(), uuidVersion7)
	}

	if parsed.String() != worktreeID {
		return fmt.Errorf("%w: %s", errNonCanonicalUUIDv7, worktreeID)
	}

	return nil
}

// Path returns the absolute path for the managed worktree ID.
func Path(root string, id string) string {
	return filepath.Join(root, id)
}
