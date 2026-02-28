package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cbout22/copilot-sync/internal/auth"
	"github.com/cbout22/copilot-sync/internal/config"
	"github.com/cbout22/copilot-sync/internal/injector"
	"github.com/cbout22/copilot-sync/internal/manifest"
	"github.com/cbout22/copilot-sync/internal/resolver"
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
	// Load the manifest
	m, err := manifest.Load(manifest.DefaultManifestFile)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	entries := m.AllEntries()
	if len(entries) == 0 {
		fmt.Println("📋 No entries in copilot.toml — nothing to sync.")
		return nil
	}

	// Load the lock file
	lock, err := manifest.LoadLock(manifest.DefaultLockFile)
	if err != nil {
		return fmt.Errorf("loading lock file: %w", err)
	}

	// Set up authenticated HTTP client
	client, err := auth.NewHTTPClient()
	if err != nil {
		return err
	}

	res := resolver.New(client)
	inj := injector.New(res, lock, ".", &injector.OSFileWriter{})

	fmt.Printf("🔄 Syncing %d asset(s)...\n\n", len(entries))

	var errors []error
	for _, entry := range entries {
		assetType := config.AssetType(entry.Type)
		fmt.Printf("  📦 %s/%s ← %s\n", entry.Type, entry.Name, entry.Ref)

		result := inj.Inject(assetType, entry.Name, entry.Ref)
		if result.Err != nil {
			fmt.Printf("  ❌ %s/%s: %s\n", entry.Type, entry.Name, result.Err)
			errors = append(errors, fmt.Errorf("%s/%s: %w", entry.Type, entry.Name, result.Err))
		} else {
			fmt.Printf("  ✅ %s/%s → %s\n", entry.Type, entry.Name, result.TargetPath)
		}
	}

	// Save the lock file
	if err := lock.Save(manifest.DefaultLockFile); err != nil {
		return fmt.Errorf("saving lock file: %w", err)
	}

	fmt.Println()
	if len(errors) > 0 {
		return fmt.Errorf("sync completed with %d error(s)", len(errors))
	}

	fmt.Println("✅ All assets synced successfully.")
	return nil
}
