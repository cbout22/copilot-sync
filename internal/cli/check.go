package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/cbout22/copilot-sync/internal/config"
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
	return runCheckWith(strict, manifest.DefaultManifestFile, manifest.DefaultLockFile, ".")
}

// runCheckWith is the testable core of the check command.
func runCheckWith(strict bool, manifestPath, lockPath, rootDir string) error {
	m, err := manifest.Load(manifestPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	entries := m.AllEntries()
	if len(entries) == 0 {
		fmt.Println("📋 No entries in copilot.toml — nothing to check.")
		return nil
	}

	lock, err := manifest.LoadLock(lockPath)
	if err != nil {
		return fmt.Errorf("loading lock file: %w", err)
	}

	fmt.Printf("🔍 Checking %d asset(s)...\n\n", len(entries))

	var issues int

	for _, entry := range entries {
		assetType := config.AssetType(entry.Type)
		targetPath := filepath.Join(rootDir, assetType.TargetPath(entry.Name))

		_, statErr := os.Stat(targetPath)
		fileExists := statErr == nil

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
			// fileExists && locked && refs match — verify content integrity
			cs, err := localChecksum(targetPath, assetType.IsDirectory())
			if err != nil {
				fmt.Printf("  ❌ %s/%s — error reading local file: %v\n", entry.Type, entry.Name, err)
				issues++
			} else if cs != lockEntry.Checksum {
				fmt.Printf("  ❌ %s/%s — content modified (checksum mismatch)\n", entry.Type, entry.Name)
				issues++
			} else {
				fmt.Printf("  ✅ %s/%s — ok\n", entry.Type, entry.Name)
			}
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

// localChecksum computes the SHA-256 checksum of a local file or directory,
// using the same algorithm as the injector for comparison against lock file entries.
func localChecksum(path string, isDir bool) (string, error) {
	if !isDir {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return manifest.Checksum(data), nil
	}

	// For directories: collect files in sorted order and concatenate content,
	// matching the deterministic algorithm used by the injector.
	type filePair struct {
		rel  string
		data []byte
	}
	var pairs []filePair
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(path, p)
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		pairs = append(pairs, filePair{rel, data})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].rel < pairs[j].rel })

	var combined []byte
	for _, p := range pairs {
		combined = append(combined, p.data...)
	}
	return manifest.Checksum(combined), nil
}
