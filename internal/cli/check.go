package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cbout22/copilot-sync/internal/injector"
	"github.com/cbout22/copilot-sync/internal/manifest"
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
	m, err := manifest.Load(manifest.DefaultManifestFile)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	entries := m.AllEntries()
	if len(entries) == 0 {
		fmt.Println("📋 No entries in copilot.toml — nothing to check.")
		return nil
	}

	lock, err := manifest.LoadLock(manifest.DefaultLockFile)
	if err != nil {
		return fmt.Errorf("loading lock file: %w", err)
	}

	fs := &injector.OSFileWriter{}
	results := CheckAssets(entries, lock, fs)

	fmt.Printf("🔍 Checking %d asset(s)...\n\n", len(results))

	var issues int
	for _, r := range results {
		switch r.Status {
		case CheckOK:
			fmt.Printf("  ✅ %s/%s — ok\n", r.Type, r.Name)
		case CheckNeverSynced:
			fmt.Printf("  ❌ %s/%s — missing (never synced)\n", r.Type, r.Name)
			issues++
		case CheckFileMissing:
			fmt.Printf("  ❌ %s/%s — missing (was synced)\n", r.Type, r.Name)
			issues++
		case CheckNotInLock:
			fmt.Printf("  ⚠️  %s/%s — file exists but not in lock file (run 'cops sync')\n", r.Type, r.Name)
			issues++
		case CheckRefMismatch:
			fmt.Printf("  ⚠️  %s/%s — ref changed: lock=%s manifest=%s\n", r.Type, r.Name, r.LockRef, r.ManifRef)
			issues++
		}
	}

	fmt.Println()
	if issues > 0 {
		msg := fmt.Sprintf("Found %d issue(s). Run 'cops sync' to fix.", issues)
		if strict {
			return fmt.Errorf("%s", msg)
		}
		fmt.Printf("⚠️  %s\n", msg)
	} else {
		fmt.Println("✅ All assets are in sync.")
	}
	return nil
}
