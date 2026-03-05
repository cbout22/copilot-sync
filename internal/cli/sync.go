package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cbout22/copilot-sync/internal/auth"
	"github.com/cbout22/copilot-sync/internal/manifest"
	"github.com/cbout22/copilot-sync/internal/port"
	"github.com/cbout22/copilot-sync/internal/resolver"
	"github.com/cbout22/copilot-sync/internal/usecase"
)

// newSyncCmd creates the `sync` command.
// Usage: cops sync
func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync all assets defined in copilot.toml",
		Long: `Downloads or updates all assets declared in copilot.toml.
Each entry is fetched from GitHub and written to its corresponding
.github/<type>/ directory.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync()
		},
	}
}

func runSync() error {
	client, err := auth.NewHTTPClient()
	if err != nil {
		return err
	}
	res := resolver.New(client)
	return runSyncWith(manifest.DefaultManifestFile, manifest.DefaultLockFile, res, ".")
}

// runSyncWith is the testable core of the sync command.
func runSyncWith(manifestPath, lockPath string, github port.GitHubResolver, rootDir string) error {
	uc := usecase.NewSyncAssets(port.OSFileSystem{}, github)

	result, err := uc.Execute(manifestPath, lockPath, rootDir)
	if err != nil {
		return err
	}

	total := len(result.Succeeded) + len(result.Failed)
	if total == 0 {
		fmt.Println("📋 No entries in copilot.toml — nothing to sync.")
		return nil
	}

	fmt.Printf("🔄 Syncing %d asset(s)...\n\n", total)

	for _, entry := range result.Succeeded {
		fmt.Printf("  ✅ %s/%s → %s\n", entry.Type, entry.Name, entry.TargetPath)
	}
	for _, entry := range result.Failed {
		fmt.Printf("  ❌ %s/%s: %s\n", entry.Type, entry.Name, entry.Err)
	}

	fmt.Println()
	if len(result.Failed) > 0 {
		return fmt.Errorf("sync completed with %d error(s)", len(result.Failed))
	}

	fmt.Println("✅ All assets synced successfully.")
	return nil
}
