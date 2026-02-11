// Verify subcommand: runs contract interactions against a live provider.
package cli

import (
	"fmt"
	"os"

	"github.com/andrewh/accord/internal/contract"
	"github.com/andrewh/accord/internal/verify"
	"github.com/spf13/cobra"
)

var providerURL string

func init() {
	verifyCmd.Flags().StringVar(&providerURL, "provider-url", "", "Base URL of the running provider")
	verifyCmd.MarkFlagRequired("provider-url")
	rootCmd.AddCommand(verifyCmd)
}

var verifyCmd = &cobra.Command{
	Use:   "verify <files...>",
	Short: "Verify contracts against a provider",
	Long:  "Sends contract interactions to a running provider and verifies the responses match.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runVerify,
}

func runVerify(cmd *cobra.Command, args []string) error {
	hasFailures := false

	for _, path := range args {
		result, err := contract.ParseFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			hasFailures = true
			continue
		}

		c := result.Contract
		fmt.Printf("Verifying %s (%s -> %s)\n", path, c.Consumer.Name, c.Provider.Name)

		results := verify.Verify(c, providerURL)
		for _, r := range results {
			if r.Passed {
				fmt.Printf("  PASS  %s\n", r.Interaction)
			} else {
				fmt.Printf("  FAIL  %s\n", r.Interaction)
				for _, f := range r.Failures {
					fmt.Printf("        %s\n", f)
				}
				hasFailures = true
			}
		}
	}

	if hasFailures {
		cmd.SilenceUsage = true
		return fmt.Errorf("verification failed")
	}
	fmt.Println("\nAll interactions passed.")
	return nil
}
