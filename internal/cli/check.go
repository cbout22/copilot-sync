package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"cops/internal/config"
	"cops/internal/manifest"
)

// newCheckCmd creates the `check` command.
// Usage: cops check [--strict]
func newCheckCmd() *cobra.Command {
	var strict bool

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check if local assets are in sync with copilot.toml",
		Long: `Validates that all entries in copilot.toml have corresponding local files
and that they match the lock file checksums. Useful in CI/CD pipelines.

With --strict, the command exits with a non-zero code if any asset is
missing or stale.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(strict)
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Exit with error code if assets are stale or missing")

	return cmd
}

func runCheck(strict bool) error {
	// Load the manifest
	m, err := manifest.Load(manifest.DefaultManifestFile)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	entries := m.AllEntries()
	if len(entries) == 0 {
		fmt.Println("📋 No entries in copilot.toml — nothing to check.")
		return nil
	}

	// Load the lock file
	lock, err := manifest.LoadLock(manifest.DefaultLockFile)
	if err != nil {
		return fmt.Errorf("loading lock file: %w", err)
	}

	fmt.Printf("🔍 Checking %d asset(s)...\n\n", len(entries))

	var issues int

	for _, entry := range entries {
		assetType := config.AssetType(entry.Type)
		targetPath := assetType.TargetPath(entry.Name)

		// Check if file exists on disk
		_, statErr := os.Stat(targetPath)
		fileExists := statErr == nil

		// Check lock file
		lockEntry, locked := lock.Get(entry.Type, entry.Name)

		switch {
		case !fileExists && !locked:
			fmt.Printf("  ❌ %s/%s — missing (never synced)\n", entry.Type, entry.Name)
			issues++
		case !fileExists && locked:
			fmt.Printf("  ❌ %s/%s — missing (was synced at %s)\n", entry.Type, entry.Name, lockEntry.SyncedAt)
			issues++
		case fileExists && !locked:
			fmt.Printf("  ⚠️  %s/%s — file exists but not in lock file (run 'cops sync')\n", entry.Type, entry.Name)
			issues++
		case fileExists && locked && lockEntry.Ref != entry.Ref:
			fmt.Printf("  ⚠️  %s/%s — ref changed: lock=%s manifest=%s\n", entry.Type, entry.Name, lockEntry.Ref, entry.Ref)
			issues++
		default:
			fmt.Printf("  ✅ %s/%s — ok\n", entry.Type, entry.Name)
		}
	}

	fmt.Println()

	if issues > 0 {
		msg := fmt.Sprintf("Found %d issue(s). Run 'cops sync' to fix.", issues)
		if strict {
			return fmt.Errorf(msg)
		}
		fmt.Printf("⚠️  %s\n", msg)
	} else {
		fmt.Println("✅ All assets are in sync.")
	}

	return nil
}
