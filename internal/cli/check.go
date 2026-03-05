package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cbout22/copilot-sync/internal/manifest"
	"github.com/cbout22/copilot-sync/internal/port"
	"github.com/cbout22/copilot-sync/internal/usecase"
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
	return runCheckWith(strict, manifest.DefaultManifestFile, manifest.DefaultLockFile, ".")
}

// runCheckWith is the testable core of the check command.
func runCheckWith(strict bool, manifestPath, lockPath, rootDir string) error {
	uc := usecase.NewCheckAssets(port.OSFileSystem{})

	result, err := uc.Execute(manifestPath, lockPath, rootDir)
	if err != nil {
		return err
	}

	if len(result.Entries) == 0 {
		fmt.Println("📋 No entries in copilot.toml — nothing to check.")
		return nil
	}

	fmt.Printf("🔍 Checking %d asset(s)...\n\n", len(result.Entries))

	for _, entry := range result.Entries {
		switch entry.Status {
		case usecase.StatusMissingNeverSynced:
			fmt.Printf("  ❌ %s/%s — missing (never synced)\n", entry.Type, entry.Name)
		case usecase.StatusMissingSynced:
			fmt.Printf("  ❌ %s/%s — missing (%s)\n", entry.Type, entry.Name, entry.Detail)
		case usecase.StatusNotInLock:
			fmt.Printf("  ⚠️  %s/%s — file exists but not in lock file (run 'cops sync')\n", entry.Type, entry.Name)
		case usecase.StatusRefChanged:
			fmt.Printf("  ⚠️  %s/%s — ref changed: %s\n", entry.Type, entry.Name, entry.Detail)
		default:
			fmt.Printf("  ✅ %s/%s — ok\n", entry.Type, entry.Name)
		}
	}

	fmt.Println()

	issues := result.Issues()
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
