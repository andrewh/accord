// Root cobra command and CLI execution entry point.
package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "accord",
	Short:         "Simple contract testing at scale",
	Long:          "Accord: Simple Contract Testing at Scale.\n\nConsumer-driven contracts using plain YAML files.",
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}
