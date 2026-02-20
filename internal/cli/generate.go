// Generate subcommand: creates contract files from an OpenAPI spec.
package cli

import (
	"fmt"

	"github.com/andrewh/accord/internal/generate"
	"github.com/spf13/cobra"
)

var (
	genConsumer  string
	genEndpoints string
	genOutputDir string
	genDryRun    bool
)

func init() {
	generateCmd.Flags().StringVar(&genConsumer, "consumer", "my-service", "Consumer service name")
	generateCmd.Flags().StringVar(&genEndpoints, "endpoints", "", "Glob pattern to filter endpoint paths")
	generateCmd.Flags().StringVar(&genOutputDir, "output-dir", ".", "Output directory for generated files")
	generateCmd.Flags().BoolVar(&genDryRun, "dry-run", false, "Print generated contracts to stdout instead of writing files")
	rootCmd.AddCommand(generateCmd)
}

var generateCmd = &cobra.Command{
	Use:   "generate <openapi-spec>",
	Short: "Generate contract files from an OpenAPI spec",
	Long:  "Reads an OpenAPI specification and generates starter Accord contract files with sensible defaults and matching rules.",
	Example: `  accord generate openapi.yaml
  accord generate --consumer checkout-service --output-dir contracts/ api/spec.yaml
  accord generate --endpoints "/users/*" --dry-run openapi.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: runGenerate,
}

func runGenerate(cmd *cobra.Command, args []string) error {
	opts := generate.Options{
		Consumer:  genConsumer,
		Endpoints: genEndpoints,
		OutputDir: genOutputDir,
	}

	outputs, err := generate.FromFile(args[0], opts)
	if err != nil {
		cmd.SilenceUsage = true
		fmt.Fprintln(cmd.ErrOrStderr(), err) //nolint:errcheck
		return err
	}

	if len(outputs) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "no interactions generated (check --endpoints filter)") //nolint:errcheck
		return nil
	}

	if genDryRun {
		stdout := cmd.OutOrStdout()
		for _, out := range outputs {
			fmt.Fprintf(stdout, "# %s\n", out.Filename) //nolint:errcheck
			fmt.Fprintln(stdout, string(out.YAML))      //nolint:errcheck
		}
		return nil
	}

	if err := generate.WriteFiles(outputs); err != nil {
		cmd.SilenceUsage = true
		return err
	}

	stdout := cmd.OutOrStdout()
	for _, out := range outputs {
		fmt.Fprintf(stdout, "wrote %s/%s\n", genOutputDir, out.Filename) //nolint:errcheck
	}
	return nil
}
