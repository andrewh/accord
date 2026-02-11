// Generate subcommand: creates contract files from an OpenAPI spec.
package cli

import (
	"fmt"
	"os"

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
	Args:  cobra.ExactArgs(1),
	RunE:  runGenerate,
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
		return err
	}

	if len(outputs) == 0 {
		fmt.Fprintln(os.Stderr, "no interactions generated (check --endpoints filter)")
		return nil
	}

	if genDryRun {
		for _, out := range outputs {
			fmt.Printf("# %s\n", out.Filename)
			fmt.Println(string(out.YAML))
		}
		return nil
	}

	if err := generate.WriteFiles(outputs); err != nil {
		cmd.SilenceUsage = true
		return err
	}

	for _, out := range outputs {
		fmt.Printf("wrote %s/%s\n", genOutputDir, out.Filename)
	}
	return nil
}
