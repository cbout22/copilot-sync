package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cbout22/copilot-sync/internal/config"
	"github.com/cbout22/copilot-sync/internal/manifest"
)

// newUnuseCmd creates the `unuse` subcommand for a given asset type.
// Usage: cops <type> unuse <name>
func newUnuseCmd(typeName string) *cobra.Command {
	return &cobra.Command{
		Use:   "unuse <name>",
		Short: fmt.Sprintf("Remove a %s entry and delete its local file", typeName),
		Long: fmt.Sprintf(`Removes a %s entry from copilot.toml and deletes the
corresponding local file or directory from disk.

Example:
  cops %s unuse my-asset`, typeName, typeName),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			return runUnuse(typeName, name)
		},
	}
}

func runUnuse(typeName, name string) error {
	assetType := config.AssetType(typeName)
	if !assetType.IsValid() {
		return fmt.Errorf("invalid asset type: %s", typeName)
	}

	// Load the manifest
	m, err := manifest.Load(manifest.DefaultManifestFile)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	// Load the lock file
	lock, err := manifest.LoadLock(manifest.DefaultLockFile)
	if err != nil {
		return fmt.Errorf("loading lock file: %w", err)
	}

	// Remove the entry
	removed, err := m.Remove(typeName, name)
	if err != nil {
		return err
	}

	if !removed {
		return fmt.Errorf("%s/%s not found in copilot.toml", typeName, name)
	}

	// Delete the local file or directory from disk
	targetPath := assetType.TargetPath(name)
	if err := os.RemoveAll(targetPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting %s: %w", targetPath, err)
	}

	// Remove from lock file
	lock.Remove(typeName, name)

	// Save the manifest
	if err := m.Save(manifest.DefaultManifestFile); err != nil {
		return fmt.Errorf("saving manifest: %w", err)
	}

	// Save the lock file
	if err := lock.Save(manifest.DefaultLockFile); err != nil {
		return fmt.Errorf("saving lock file: %w", err)
	}

	fmt.Printf("🗑️  Removed %s/%s from copilot.toml\n", typeName, name)
	fmt.Printf("🧹 Deleted %s\n", targetPath)
	return nil
}
