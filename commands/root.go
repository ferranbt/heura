package commands

import (
	"github.com/spf13/cobra"

	"github.com/umbracle/heura/commands/repl"
	"github.com/umbracle/heura/commands/run"
	"github.com/umbracle/heura/commands/version"
)

var rootCmd = &cobra.Command{
	Use: "heura",
}

func init() {
	rootCmd.AddCommand(
		repl.RootCmd,
		run.RootCmd,
		version.RootCmd,
	)
}

// Command returns the root command
func Command() *cobra.Command {
	return rootCmd
}
