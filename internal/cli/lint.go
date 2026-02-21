// Lint subcommand: validates contract files and reports diagnostics.
package cli

import (
	"fmt"

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
	Example: `  accord lint contracts/user-api.yaml
  accord lint contracts/*.yaml
  accord lint "contracts/**/*.yaml"
  accord lint contracts/`,
	Args: cobra.MinimumNArgs(1),
	RunE: runLint,
}

func runLint(cmd *cobra.Command, args []string) error {
	paths, err := resolveFilePaths(args)
	if err != nil {
		cmd.SilenceUsage = true
		return err
	}

	linter := lint.New()
	hasErrors := false

	for _, path := range paths {
		result, err := contract.ParseFile(path)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", path, err) //nolint:errcheck
			hasErrors = true
			continue
		}

		diags := linter.Lint(result.Contract, result.Node)
		for _, d := range diags {
			fmt.Fprintln(cmd.OutOrStdout(), lint.FormatDiagnostic(path, d)) //nolint:errcheck
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
