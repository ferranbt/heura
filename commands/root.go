package commands

import (
	"github.com/spf13/cobra"

	"github.com/umbracle/heura/commands/repl"
	"github.com/umbracle/heura/commands/run"
)

var rootCmd = &cobra.Command{
	Use: "heura",
}

func init() {
	rootCmd.AddCommand(
		repl.RootCmd,
		run.RootCmd,
	)
}

// Command returns the root command
func Command() *cobra.Command {
	return rootCmd
}
