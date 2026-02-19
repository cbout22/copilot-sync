package cli

import (
	"strings"

	"github.com/cbout22/copilot-sync/internal/manifest"
	"github.com/spf13/cobra"
)

func resolveManifestName(assetType string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Load the manifest
	m, err := manifest.Load(manifest.DefaultManifestFile)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get all asset names for the given type
	names, err := m.Section(assetType)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Filter based on toComplete prefix
	var completions []string
	for name, ref := range names {
		if strings.HasPrefix(name, toComplete) {
			completions = append(completions, formatCompletionLine(name, ref))
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
