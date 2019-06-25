package version

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/umbracle/heura/version"
)

// RootCmd returns the repl command
var RootCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Heura version",
	Run:   rootRun,
}

func rootRun(cmd *cobra.Command, args []string) {
	fmt.Printf("Heura %s", version.GetVersion())
}
