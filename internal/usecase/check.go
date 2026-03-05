package usecase

import (
	"fmt"
	"path/filepath"

	"github.com/cbout22/copilot-sync/internal/config"
	"github.com/cbout22/copilot-sync/internal/manifest"
	"github.com/cbout22/copilot-sync/internal/port"
)

// CheckAssets orchestrates checking all assets against the manifest and lock file.
type CheckAssets struct {
	fs port.FileSystem
}

// CheckStatus represents the sync status of an asset.
type CheckStatus int

const (
	StatusOK CheckStatus = iota
	StatusMissingNeverSynced
	StatusMissingSynced
	StatusNotInLock
	StatusRefChanged
)

// CheckEntry records the check result for a single asset.
type CheckEntry struct {
	Type   string
	Name   string
	Status CheckStatus
	Detail string
}

// CheckResult holds the outcome of a check operation.
type CheckResult struct {
	Entries []CheckEntry
}

// Issues returns the number of entries that are not OK.
func (r *CheckResult) Issues() int {
	count := 0
	for _, e := range r.Entries {
		if e.Status != StatusOK {
			count++
		}
	}
	return count
}

// NewCheckAssets creates a CheckAssets use case.
func NewCheckAssets(fs port.FileSystem) *CheckAssets {
	return &CheckAssets{fs: fs}
}

// Execute checks all assets against the manifest and lock file.
func (c *CheckAssets) Execute(manifestPath, lockPath, rootDir string) (*CheckResult, error) {
	m, err := manifest.LoadWith(manifestPath, c.fs)
	if err != nil {
		return nil, fmt.Errorf("loading manifest: %w", err)
	}

	entries := m.AllEntries()
	if len(entries) == 0 {
		return &CheckResult{}, nil
	}

	lock, err := manifest.LoadLockWith(lockPath, c.fs)
	if err != nil {
		return nil, fmt.Errorf("loading lock file: %w", err)
	}

	result := &CheckResult{}
	for _, entry := range entries {
		assetType := config.AssetType(entry.Type)
		targetPath := filepath.Join(rootDir, assetType.TargetPath(entry.Name))

		_, statErr := c.fs.Stat(targetPath)
		fileExists := statErr == nil

		lockEntry, locked := lock.Get(entry.Type, entry.Name)

		var ce CheckEntry
		ce.Type = entry.Type
		ce.Name = entry.Name

		switch {
		case !fileExists && !locked:
			ce.Status = StatusMissingNeverSynced
		case !fileExists && locked:
			ce.Status = StatusMissingSynced
			ce.Detail = fmt.Sprintf("was synced at %s", lockEntry.SyncedAt)
		case fileExists && !locked:
			ce.Status = StatusNotInLock
		case fileExists && locked && lockEntry.Ref != entry.Ref:
			ce.Status = StatusRefChanged
			ce.Detail = fmt.Sprintf("lock=%s manifest=%s", lockEntry.Ref, entry.Ref)
		default:
			ce.Status = StatusOK
		}

		result.Entries = append(result.Entries, ce)
	}

	return result, nil
}
