// Root cobra command and CLI execution entry point.
package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "accord",
	Short:         "File-based contract testing",
	Long:          "Accord is a consumer-driven contract testing tool using plain YAML files.",
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}
