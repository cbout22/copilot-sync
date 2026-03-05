package usecase

import (
	"fmt"
	"path/filepath"

	"github.com/cbout22/copilot-sync/internal/config"
	"github.com/cbout22/copilot-sync/internal/manifest"
	"github.com/cbout22/copilot-sync/internal/port"
)

// UnuseAsset orchestrates removing an asset from the manifest, deleting its file, and updating the lock.
type UnuseAsset struct {
	fs port.FileSystem
}

// NewUnuseAsset creates an UnuseAsset use case.
func NewUnuseAsset(fs port.FileSystem) *UnuseAsset {
	return &UnuseAsset{fs: fs}
}

// Execute removes an asset from the manifest, deletes its file, and updates the lock.
func (u *UnuseAsset) Execute(typeName, name, manifestPath, lockPath, rootDir string) error {
	assetType := config.AssetType(typeName)
	if !assetType.IsValid() {
		return fmt.Errorf("invalid asset type: %s", typeName)
	}

	m, err := manifest.LoadWith(manifestPath, u.fs)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	lock, err := manifest.LoadLockWith(lockPath, u.fs)
	if err != nil {
		return fmt.Errorf("loading lock file: %w", err)
	}

	removed, err := m.Remove(typeName, name)
	if err != nil {
		return err
	}
	if !removed {
		return fmt.Errorf("%s/%s not found in copilot.toml", typeName, name)
	}

	targetPath := filepath.Join(rootDir, assetType.TargetPath(name))
	if err := u.fs.RemoveAll(targetPath); err != nil {
		return fmt.Errorf("deleting %s: %w", targetPath, err)
	}

	lock.Remove(typeName, name)

	if err := m.SaveWith(manifestPath, u.fs); err != nil {
		return fmt.Errorf("saving manifest: %w", err)
	}

	if err := lock.SaveWith(lockPath, u.fs); err != nil {
		return fmt.Errorf("saving lock file: %w", err)
	}

	return nil
}
