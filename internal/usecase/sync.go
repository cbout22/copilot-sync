package usecase

import (
	"fmt"

	"github.com/cbout22/copilot-sync/internal/config"
	"github.com/cbout22/copilot-sync/internal/injector"
	"github.com/cbout22/copilot-sync/internal/manifest"
	"github.com/cbout22/copilot-sync/internal/port"
)

// SyncAssets orchestrates syncing all assets declared in the manifest.
type SyncAssets struct {
	fs     port.FileSystem
	github port.GitHubResolver
}

// SyncEntry records the result of syncing a single asset.
type SyncEntry struct {
	Type       string
	Name       string
	Ref        string
	TargetPath string
	Err        error
}

// SyncResult holds the outcome of a sync operation.
type SyncResult struct {
	Succeeded []SyncEntry
	Failed    []SyncEntry
}

// NewSyncAssets creates a SyncAssets use case.
func NewSyncAssets(fs port.FileSystem, github port.GitHubResolver) *SyncAssets {
	return &SyncAssets{fs: fs, github: github}
}

// Execute syncs all assets declared in the manifest.
func (s *SyncAssets) Execute(manifestPath, lockPath, rootDir string) (*SyncResult, error) {
	m, err := manifest.LoadWith(manifestPath, s.fs)
	if err != nil {
		return nil, fmt.Errorf("loading manifest: %w", err)
	}

	entries := m.AllEntries()
	if len(entries) == 0 {
		return &SyncResult{}, nil
	}

	lock, err := manifest.LoadLockWith(lockPath, s.fs)
	if err != nil {
		return nil, fmt.Errorf("loading lock file: %w", err)
	}

	inj := injector.New(s.github, s.fs, lock, rootDir)

	result := &SyncResult{}
	for _, entry := range entries {
		assetType := config.AssetType(entry.Type)
		injResult := inj.Inject(assetType, entry.Name, entry.Ref)

		se := SyncEntry{
			Type:       entry.Type,
			Name:       entry.Name,
			Ref:        entry.Ref,
			TargetPath: injResult.TargetPath,
			Err:        injResult.Err,
		}

		if injResult.Err != nil {
			result.Failed = append(result.Failed, se)
		} else {
			result.Succeeded = append(result.Succeeded, se)
		}
	}

	if err := lock.SaveWith(lockPath, s.fs); err != nil {
		return nil, fmt.Errorf("saving lock file: %w", err)
	}

	return result, nil
}
