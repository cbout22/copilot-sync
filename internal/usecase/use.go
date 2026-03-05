package usecase

import (
	"fmt"

	"github.com/cbout22/copilot-sync/internal/config"
	"github.com/cbout22/copilot-sync/internal/injector"
	"github.com/cbout22/copilot-sync/internal/manifest"
	"github.com/cbout22/copilot-sync/internal/port"
)

// UseAsset orchestrates adding an asset to the manifest, downloading it, and updating the lock.
type UseAsset struct {
	fs     port.FileSystem
	github port.GitHubResolver
}

// UseResult holds the outcome of a use operation.
type UseResult struct {
	TargetPath string
}

// NewUseAsset creates a UseAsset use case.
func NewUseAsset(fs port.FileSystem, github port.GitHubResolver) *UseAsset {
	return &UseAsset{fs: fs, github: github}
}

// Execute adds an asset to the manifest, downloads it, and updates the lock.
func (u *UseAsset) Execute(typeName, name, rawRef, manifestPath, lockPath, rootDir string) (*UseResult, error) {
	assetType := config.AssetType(typeName)
	if !assetType.IsValid() {
		return nil, fmt.Errorf("invalid asset type: %s", typeName)
	}

	if _, err := config.ParseRef(rawRef); err != nil {
		return nil, err
	}

	m, err := manifest.LoadWith(manifestPath, u.fs)
	if err != nil {
		return nil, fmt.Errorf("loading manifest: %w", err)
	}

	lock, err := manifest.LoadLockWith(lockPath, u.fs)
	if err != nil {
		return nil, fmt.Errorf("loading lock file: %w", err)
	}

	inj := injector.New(u.github, u.fs, lock, rootDir)

	result := inj.Inject(assetType, name, rawRef)
	if result.Err != nil {
		return nil, fmt.Errorf("failed to download: %w", result.Err)
	}

	if err := m.Set(typeName, name, rawRef); err != nil {
		return nil, err
	}

	if err := m.SaveWith(manifestPath, u.fs); err != nil {
		return nil, fmt.Errorf("saving manifest: %w", err)
	}

	if err := lock.SaveWith(lockPath, u.fs); err != nil {
		return nil, fmt.Errorf("saving lock file: %w", err)
	}

	return &UseResult{TargetPath: result.TargetPath}, nil
}
