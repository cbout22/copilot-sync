package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags.
var version = "dev"

// NewRootCmd creates the top-level `cops` command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "cops",
		Short: "Copilot Sync — the deterministic package manager for your copilot AI agent files",
		Long: `cops manages your GitHub Copilot assets (instructions, agents, prompts, skills)
through a copilot.toml manifest. Pin your team's practices to a specific release
tag, branch, or commit hash and sync them across projects.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Register type subcommands (instructions, agents, prompts, skills)
	root.AddCommand(newTypeCmd("instructions", "Manage instruction files"))
	root.AddCommand(newTypeCmd("agents", "Manage agent files"))
	root.AddCommand(newTypeCmd("prompts", "Manage prompt files"))
	root.AddCommand(newTypeCmd("skills", "Manage skill directories"))

	// Register top-level commands
	root.AddCommand(newSyncCmd())
	root.AddCommand(newCheckCmd())

	return root
}

// Execute runs the root command.
func Execute() {
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
