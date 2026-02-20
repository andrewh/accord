// Convert subcommand: imports Pact v2/v3 contracts to Accord YAML.
package cli

import (
	"fmt"

	"github.com/andrewh/accord/internal/convert"
	"github.com/spf13/cobra"
)

var (
	convertOutputDir string
	convertDryRun    bool
)

func init() {
	convertCmd.Flags().StringVar(&convertOutputDir, "output-dir", ".", "Output directory for converted files")
	convertCmd.Flags().BoolVar(&convertDryRun, "dry-run", false, "Print converted contracts to stdout instead of writing files")
	rootCmd.AddCommand(convertCmd)
}

var convertCmd = &cobra.Command{
	Use:   "convert <pact-files...>",
	Short: "Convert Pact contracts to Accord YAML",
	Long:  "Reads Pact v2 or v3 JSON contract files and converts them to Accord YAML format.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runConvert,
}

func runConvert(cmd *cobra.Command, args []string) error {
	opts := convert.Options{
		OutputDir: convertOutputDir,
	}

	hasErrors := false
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	for _, path := range args {
		outputs, warnings, err := convert.FromFile(path, opts)
		if err != nil {
			cmd.SilenceUsage = true
			fmt.Fprintf(stderr, "error: %s: %v\n", path, err) //nolint:errcheck
			hasErrors = true
			continue
		}

		for _, w := range warnings {
			fmt.Fprintf(stderr, "warning: %s: %s\n", w.File, w.Message) //nolint:errcheck
		}

		if convertDryRun {
			for _, out := range outputs {
				fmt.Fprintf(stdout, "# %s\n", out.Filename) //nolint:errcheck
				fmt.Fprintln(stdout, string(out.YAML))      //nolint:errcheck
			}
			continue
		}

		if err := convert.WriteFiles(outputs); err != nil {
			cmd.SilenceUsage = true
			fmt.Fprintf(stderr, "error: %v\n", err) //nolint:errcheck
			hasErrors = true
			continue
		}

		for _, out := range outputs {
			fmt.Fprintf(stdout, "wrote %s/%s\n", convertOutputDir, out.Filename) //nolint:errcheck
		}
	}

	if hasErrors {
		return fmt.Errorf("one or more files failed to convert")
	}
	return nil
}
