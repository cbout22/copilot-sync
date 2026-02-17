package cli

import (
	"github.com/spf13/cobra"
)

// newTypeCmd creates a subcommand for a given asset type (instructions, agents, prompts, skills).
// Each type command has `use` and `unuse` sub-subcommands.
func newTypeCmd(typeName, description string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   typeName,
		Short: description,
		Long:  description + ". Use the 'use' and 'unuse' subcommands to manage entries.",
	}

	cmd.AddCommand(newUseCmd(typeName))
	cmd.AddCommand(newUnuseCmd(typeName))

	return cmd
}
