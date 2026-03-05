package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cbout22/copilot-sync/internal/config"
	"github.com/cbout22/copilot-sync/internal/manifest"
	"github.com/cbout22/copilot-sync/internal/port"
	"github.com/cbout22/copilot-sync/internal/usecase"
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
	uc := usecase.NewUnuseAsset(port.OSFileSystem{})

	if err := uc.Execute(typeName, name, manifestPath, lockPath, rootDir); err != nil {
		return err
	}

	assetType := config.AssetType(typeName)
	fmt.Printf("🗑️  Removed %s/%s from copilot.toml\n", typeName, name)
	fmt.Printf("🧹 Deleted %s\n", assetType.TargetPath(name))
	return nil
}
