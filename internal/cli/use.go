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

// newUseCmd creates the `use` subcommand for a given asset type.
// Usage: cops <type> use <name> <org/repo/path@ref>
func newUseCmd(typeName string) *cobra.Command {
	return &cobra.Command{
		Use:   "use <name> <org/repo/path@ref>",
		Short: fmt.Sprintf("Add a %s entry and download it", typeName),
		Long: fmt.Sprintf(`Adds a %s entry to copilot.toml and downloads the file from GitHub.

Example:
  cops %s use my-asset my-org/repo/path/to/file@v1.0`, typeName, typeName),
		Args: cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 1 {
				return resolveGitHubCompletions(toComplete)
			}
			return nil, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			rawRef := args[1]

			return runUse(typeName, name, rawRef)
		},
	}
}

func runUse(typeName, name, rawRef string) error {
	assetType := config.AssetType(typeName)
	if !assetType.IsValid() {
		return fmt.Errorf("invalid asset type: %s", typeName)
	}

	// Validate the ref format early
	if _, err := config.ParseRef(rawRef); err != nil {
		return err
	}

	// Load or create the manifest
	m, err := manifest.Load(manifest.DefaultManifestFile)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
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

	// Create resolver and injector
	res := resolver.New(client)
	inj := injector.New(res, lock, ".", &injector.OSFileWriter{})

	fmt.Printf("📦 Adding %s/%s from %s...\n", typeName, name, rawRef)

	// Download and inject the asset
	result := inj.Inject(assetType, name, rawRef)
	if result.Err != nil {
		return fmt.Errorf("failed to download: %w", result.Err)
	}

	// Update the manifest
	if err := m.Set(typeName, name, rawRef); err != nil {
		return err
	}

	// Save the manifest
	if err := m.Save(manifest.DefaultManifestFile); err != nil {
		return fmt.Errorf("saving manifest: %w", err)
	}

	// Save the lock file
	if err := lock.Save(manifest.DefaultLockFile); err != nil {
		return fmt.Errorf("saving lock file: %w", err)
	}

	fmt.Printf("✅ %s/%s synced to %s\n", typeName, name, result.TargetPath)
	return nil
}
