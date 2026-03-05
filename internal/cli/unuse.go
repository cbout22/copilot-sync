package cli

import (
	"fmt"
	"os"
	"path/filepath"

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
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return resolveManifestName(typeName, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			return runUnuse(typeName, name)
		},
	}
}

func runUnuse(typeName, name string) error {
	return runUnuseWith(typeName, name, manifest.DefaultManifestFile, manifest.DefaultLockFile, ".")
}

// runUnuseWith is the testable core of the unuse command.
func runUnuseWith(typeName, name, manifestPath, lockPath, rootDir string) error {
	assetType := config.AssetType(typeName)
	if !assetType.IsValid() {
		return fmt.Errorf("invalid asset type: %s", typeName)
	}

	// Load the manifest
	m, err := manifest.Load(manifestPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	// Load the lock file
	lock, err := manifest.LoadLock(lockPath)
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
	targetPath := filepath.Join(rootDir, assetType.TargetPath(name))
	if err := os.RemoveAll(targetPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting %s: %w", targetPath, err)
	}

	// Remove from lock file
	lock.Remove(typeName, name)

	// Save the manifest
	if err := m.Save(manifestPath); err != nil {
		return fmt.Errorf("saving manifest: %w", err)
	}

	// Save the lock file
	if err := lock.Save(lockPath); err != nil {
		return fmt.Errorf("saving lock file: %w", err)
	}

	fmt.Printf("🗑️  Removed %s/%s from copilot.toml\n", typeName, name)
	fmt.Printf("🧹 Deleted %s\n", assetType.TargetPath(name))
	return nil
}
