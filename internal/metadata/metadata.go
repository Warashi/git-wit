// Package metadata stores git-wit metadata blobs and refs.
package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Warashi/git-wit/internal/git"
)

const (
	expectedRefRecordFieldCount = 2
	refPrefix                   = "refs/git-wit/"
	schemaVersion               = "1.0"
)

var errEmptyObjectID = errors.New("empty object id")

// Metadata represents the JSON document stored in git blobs.
type Metadata struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"` //nolint:tagliatelle // External JSON schema is fixed by design.md.
	Memo      string    `json:"memo"`
	Version   string    `json:"version"`
}

type refRecord struct {
	id   string
	hash string
}

// New creates a metadata document for a new worktree.
func New(id string, now time.Time, memo string) Metadata {
	return Metadata{
		ID:        id,
		CreatedAt: now.UTC(),
		Memo:      memo,
		Version:   schemaVersion,
	}
}

// Store writes the metadata blob and updates refs/git-wit/<id>.
func Store(ctx context.Context, runner git.Runner, item Metadata) error {
	payload, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	result, err := runner.RunInput(ctx, string(payload), "hash-object", "-w", "--stdin")
	if err != nil {
		return fmt.Errorf("create metadata blob: %w", err)
	}

	if result.Stdout == "" {
		return fmt.Errorf("create metadata blob: %w", errEmptyObjectID)
	}

	_, err = runner.Run(ctx, "update-ref", refPrefix+item.ID, result.Stdout)
	if err != nil {
		return fmt.Errorf("update metadata ref: %w", err)
	}

	return nil
}

// Load returns the metadata stored for an ID.
func Load(ctx context.Context, runner git.Runner, id string) (Metadata, error) {
	result, err := runner.Run(ctx, "cat-file", "-p", refPrefix+id)
	if err != nil {
		return Metadata{}, fmt.Errorf("read metadata blob: %w", err)
	}

	item, err := decode(result.Stdout)
	if err != nil {
		return Metadata{}, err
	}

	return item, nil
}

// Delete removes the metadata ref for an ID.
func Delete(ctx context.Context, runner git.Runner, id string) error {
	_, err := runner.Run(ctx, "update-ref", "-d", refPrefix+id)
	if err != nil {
		return fmt.Errorf("delete metadata ref: %w", err)
	}

	return nil
}

// Exists reports whether refs/git-wit/<id> exists.
func Exists(ctx context.Context, runner git.Runner, id string) (bool, error) {
	_, err := runner.Run(ctx, "show-ref", "--verify", "--quiet", refPrefix+id)
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
func List(ctx context.Context, runner git.Runner) ([]Metadata, error) {
	result, err := runner.Run(ctx, "for-each-ref", "--format=%(refname:strip=2) %(objectname)", refPrefix)
	if err != nil {
		return nil, fmt.Errorf("list metadata refs: %w", err)
	}

	if result.Stdout == "" {
		return []Metadata{}, nil
	}

	records := parseRefRecords(result.Stdout)

	items := make([]Metadata, 0, len(records))
	for _, record := range records {
		blobResult, blobErr := runner.Run(ctx, "cat-file", "-p", record.hash)
		if blobErr != nil {
			return nil, fmt.Errorf("read metadata blob for %s: %w", record.id, blobErr)
		}

		item, decodeErr := decode(blobResult.Stdout)
		if decodeErr != nil {
			return nil, decodeErr
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

func decode(raw string) (Metadata, error) {
	var item Metadata

	err := json.Unmarshal([]byte(raw), &item)
	if err != nil {
		return Metadata{}, fmt.Errorf("decode metadata: %w", err)
	}

	return item, nil
}

func parseRefRecords(stdout string) []refRecord {
	lines := strings.Split(stdout, "\n")

	records := make([]refRecord, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != expectedRefRecordFieldCount {
			continue
		}

		records = append(records, refRecord{
			id:   strings.TrimPrefix(fields[0], "git-wit/"),
			hash: fields[1],
		})
	}

	return records
}
