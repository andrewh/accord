// Lint subcommand: validates contract files and reports diagnostics.
package cli

import (
	"fmt"
	"os"

	"github.com/andrewh/accord/internal/contract"
	"github.com/andrewh/accord/internal/lint"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(lintCmd)
}

var lintCmd = &cobra.Command{
	Use:   "lint <files...>",
	Short: "Validate contract files",
	Long:  "Validates one or more Accord contract files and reports errors and warnings.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runLint,
}

func runLint(cmd *cobra.Command, args []string) error {
	linter := lint.New()
	hasErrors := false

	for _, path := range args {
		result, err := contract.ParseFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			hasErrors = true
			continue
		}

		diags := linter.Lint(result.Contract, result.Node)
		for _, d := range diags {
			fmt.Println(lint.FormatDiagnostic(path, d))
		}
		if lint.HasErrors(diags) {
			hasErrors = true
		}
	}

	if hasErrors {
		cmd.SilenceUsage = true
		return fmt.Errorf("lint found errors")
	}
	return nil
}
