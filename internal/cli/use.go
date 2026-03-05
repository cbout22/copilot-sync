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
	client, err := auth.NewHTTPClient()
	if err != nil {
		return err
	}
	res := resolver.New(client)
	return runUseWith(typeName, name, rawRef, manifest.DefaultManifestFile, manifest.DefaultLockFile, res, ".")
}

// runUseWith is the testable core of the use command.
func runUseWith(typeName, name, rawRef, manifestPath, lockPath string, github port.GitHubResolver, rootDir string) error {
	uc := usecase.NewUseAsset(port.OSFileSystem{}, github)

	fmt.Printf("📦 Adding %s/%s from %s...\n", typeName, name, rawRef)

	result, err := uc.Execute(typeName, name, rawRef, manifestPath, lockPath, rootDir)
	if err != nil {
		return err
	}

	fmt.Printf("✅ %s/%s synced to %s\n", typeName, name, result.TargetPath)
	return nil
}
