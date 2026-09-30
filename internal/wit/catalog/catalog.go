// Package catalog owns managed worktree identity, metadata refs, and repository context.
package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Warashi/git-wit/internal/git"
	"github.com/google/uuid"
)

const (
	expectedRefRecordFieldCount = 2
	refPrefix                   = "refs/git-wit/"
	schemaVersion               = "1.1"
	uuidVersion7                = 7
	worktreeRootPerm            = 0o750
)

var (
	errEmptyObjectID        = errors.New("empty object id")
	errEmptyWorktreeRoot    = errors.New("empty worktree root")
	errInvalidUUIDVersion   = errors.New("invalid uuid version")
	errNonCanonicalUUIDv7   = errors.New("non-canonical uuidv7")
	errNotManagedWorktree   = errors.New("not in a managed worktree")
	errRelativeWorktreeRoot = errors.New("wit.worktree.root must be an absolute path")
)

// Record represents the JSON document stored in git blobs.
type Record struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"` //nolint:tagliatelle // External JSON schema is fixed by design.md.
	Memo      string    `json:"memo"`
	Base      string    `json:"base"`
	Version   string    `json:"version"`
}

type refRecord struct {
	id   string
	hash string
}

// Repository describes the resolved git-wit repository context.
type Repository struct {
	root         string
	runner       git.Runner
	worktreeRoot string
}

// NewID returns a new lexicographically sortable worktree identifier.
func NewID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// NewRecord creates a metadata document for a new worktree. base is the
// commit the worktree is created at; read models use it to tell "work was
// integrated" apart from "no work happened yet".
func NewRecord(now time.Time, memo string, base string) Record {
	return Record{
		ID:        NewID(),
		CreatedAt: now.UTC(),
		Memo:      memo,
		Base:      base,
		Version:   schemaVersion,
	}
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

// Open resolves the current repository from cwd.
func Open(ctx context.Context, cwd string) (Repository, error) {
	runner := git.NewRunner(cwd)

	result, err := runner.Run(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return Repository{}, fmt.Errorf("discover repository root: %w", err)
	}

	root, err := filepath.Abs(result.Stdout)
	if err != nil {
		return Repository{}, fmt.Errorf("normalize repository root: %w", err)
	}

	repoRunner := git.NewRunner(root)

	worktreeRoot, err := loadWorktreeRoot(ctx, repoRunner)
	if err != nil {
		return Repository{}, fmt.Errorf("load worktree root: %w", err)
	}

	// git reports physical (symlink-resolved) paths, while the configured
	// root is taken verbatim; resolve it so path comparisons like
	// CurrentID agree with what rev-parse returns.
	worktreeRoot = resolveSymlinksBestEffort(worktreeRoot)

	return Repository{
		root:         root,
		runner:       repoRunner,
		worktreeRoot: worktreeRoot,
	}, nil
}

// resolveSymlinksBestEffort resolves symlinks in path even when its deepest
// components do not exist yet (the worktree root is created lazily), by
// resolving the nearest existing ancestor and re-appending the remainder.
func resolveSymlinksBestEffort(path string) string {
	remainder := ""
	current := path

	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			return filepath.Join(resolved, remainder)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return path
		}

		remainder = filepath.Join(filepath.Base(current), remainder)
		current = parent
	}
}

// Root returns the absolute repository root.
func (r Repository) Root() string {
	return r.root
}

// Runner returns a git runner rooted at the repository.
func (r Repository) Runner() git.Runner {
	return r.runner
}

// WorktreeRoot returns the managed worktree directory.
func (r Repository) WorktreeRoot() string {
	return r.worktreeRoot
}

// WorktreePath returns the path for a managed worktree ID.
func (r Repository) WorktreePath(id string) string {
	return filepath.Join(r.worktreeRoot, id)
}

// EnsureWorktreeRoot creates the managed worktree root if needed.
func (r Repository) EnsureWorktreeRoot() error {
	err := os.MkdirAll(r.worktreeRoot, worktreeRootPerm)
	if err != nil {
		return fmt.Errorf("create worktree root: %w", err)
	}

	return nil
}

// Store writes the metadata blob and updates refs/git-wit/<id>.
func (r Repository) Store(ctx context.Context, record Record) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	result, err := r.runner.RunInput(ctx, string(payload), "hash-object", "-w", "--stdin")
	if err != nil {
		return fmt.Errorf("create metadata blob: %w", err)
	}

	if result.Stdout == "" {
		return fmt.Errorf("create metadata blob: %w", errEmptyObjectID)
	}

	_, err = r.runner.Run(ctx, "update-ref", refPrefix+record.ID, result.Stdout)
	if err != nil {
		return fmt.Errorf("update metadata ref: %w", err)
	}

	return nil
}

// Load returns the metadata stored for an ID.
func (r Repository) Load(ctx context.Context, id string) (Record, error) {
	result, err := r.runner.Run(ctx, "cat-file", "-p", refPrefix+id)
	if err != nil {
		return Record{}, fmt.Errorf("read metadata blob: %w", err)
	}

	record, err := decode(result.Stdout)
	if err != nil {
		return Record{}, err
	}

	return record, nil
}

// Delete removes the metadata ref for an ID.
func (r Repository) Delete(ctx context.Context, id string) error {
	_, err := r.runner.Run(ctx, "update-ref", "-d", refPrefix+id)
	if err != nil {
		return fmt.Errorf("delete metadata ref: %w", err)
	}

	return nil
}

// Exists reports whether refs/git-wit/<id> exists.
func (r Repository) Exists(ctx context.Context, id string) (bool, error) {
	_, err := r.runner.Run(ctx, "show-ref", "--verify", "--quiet", refPrefix+id)
	if err == nil {
		return true, nil
	}

	var cmdErr git.CommandError
	if !errors.As(err, &cmdErr) {
		return false, fmt.Errorf("check metadata ref: %w", err)
	}

	if cmdErr.ExitCode() == 1 {
		return false, nil
	}

	return false, fmt.Errorf("check metadata ref: %w", err)
}

// List returns all git-wit metadata documents sorted by creation time, then ID.
func (r Repository) List(ctx context.Context) ([]Record, error) {
	result, err := r.runner.Run(ctx, "for-each-ref", "--format=%(refname:strip=2) %(objectname)", refPrefix)
	if err != nil {
		return nil, fmt.Errorf("list metadata refs: %w", err)
	}

	if result.Stdout == "" {
		return []Record{}, nil
	}

	records := parseRefRecords(result.Stdout)

	// Corrupt or foreign entries under refs/git-wit/ are skipped rather
	// than failing the whole listing: List backs ls, prune, and rm
	// --merged, and failing here would leave no built-in way to recover.
	// A skipped entry's directory still surfaces via prune as an orphan.
	items := make([]Record, 0, len(records))
	for _, record := range records {
		blobResult, blobErr := r.runner.Run(ctx, "cat-file", "-p", record.hash)
		if blobErr != nil {
			continue
		}

		item, decodeErr := decode(blobResult.Stdout)
		if decodeErr != nil {
			continue
		}

		// A blob whose id disagrees with its ref name would make callers
		// act (stat, prune, remove) on a path or ref they never listed.
		if item.ID != record.id {
			continue
		}

		items = append(items, item)
	}

	sort.Slice(items, func(i int, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}

		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})

	return items, nil
}

// CurrentID returns the managed worktree ID for the active repository context.
func (r Repository) CurrentID() (string, error) {
	rel, err := filepath.Rel(r.worktreeRoot, r.root)
	if err != nil {
		return "", fmt.Errorf("rel worktree path: %w", err)
	}

	if rel == "." || filepath.Dir(rel) != "." {
		return "", errNotManagedWorktree
	}

	err = ValidateID(rel)
	if err != nil {
		return "", fmt.Errorf("validate worktree id: %w", err)
	}

	return rel, nil
}

func isMissingConfig(err error, target *git.CommandError) bool {
	if !strings.Contains(err.Error(), " config ") {
		return false
	}

	if !errors.As(err, target) {
		return false
	}

	return target.ExitCode() == 1
}

func decode(raw string) (Record, error) {
	var record Record

	err := json.Unmarshal([]byte(raw), &record)
	if err != nil {
		return Record{}, fmt.Errorf("decode metadata: %w", err)
	}

	return record, nil
}

func parseRefRecords(stdout string) []refRecord {
	lines := strings.Split(stdout, "\n")

	records := make([]refRecord, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != expectedRefRecordFieldCount {
			continue
		}

		recordID := strings.TrimPrefix(fields[0], "git-wit/")
		if ValidateID(recordID) != nil {
			continue
		}

		records = append(records, refRecord{
			id:   recordID,
			hash: fields[1],
		})
	}

	return records
}

func loadWorktreeRoot(ctx context.Context, runner git.Runner) (string, error) {
	result, err := runner.Run(ctx, "config", "--get", "wit.worktree.root")
	if err != nil {
		var cmdErr git.CommandError
		if isMissingConfig(err, &cmdErr) {
			return defaultWorktreeRoot()
		}

		return "", fmt.Errorf("load wit.worktree.root: %w", err)
	}

	root := strings.TrimSpace(result.Stdout)
	if root == "" {
		return "", fmt.Errorf("load wit.worktree.root: %w", errEmptyWorktreeRoot)
	}

	// A relative root would resolve against the process working
	// directory, so the same repository would use a different root per
	// invocation and prune would treat every worktree as missing.
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("load wit.worktree.root: %w: %s", errRelativeWorktreeRoot, root)
	}

	return filepath.Clean(root), nil
}

func defaultWorktreeRoot() (string, error) {
	baseDir := os.Getenv("XDG_STATE_HOME")

	// The XDG base directory spec requires ignoring relative paths, and
	// honoring one here would inherit the same cwd-dependence rejected
	// for wit.worktree.root above.
	if !filepath.IsAbs(baseDir) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}

		baseDir = filepath.Join(homeDir, ".local", "state")
	}

	return filepath.Join(baseDir, "git-wit", "worktrees"), nil
}
