// Verify subcommand: runs contract interactions against a live provider.
package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/andrewh/accord/internal/contract"
	"github.com/andrewh/accord/internal/verify"
	"github.com/spf13/cobra"
)

var providerURL string
var timeout int

func init() {
	verifyCmd.Flags().StringVar(&providerURL, "provider-url", "", "Base URL of the running provider")
	verifyCmd.Flags().IntVar(&timeout, "timeout", 30, "HTTP request timeout in seconds")
	_ = verifyCmd.MarkFlagRequired("provider-url")
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

		results := verify.Verify(c, providerURL, time.Duration(timeout)*time.Second)
		for _, r := range results {
			hasWarnings := len(r.Failures) > 0 && r.Passed
			switch {
			case r.Passed && !hasWarnings:
				fmt.Printf("  PASS  %s\n", r.Interaction)
			case r.Passed && hasWarnings:
				fmt.Printf("  WARN  %s\n", r.Interaction)
				for _, f := range r.Failures {
					fmt.Printf("        [warning] %s\n", f)
				}
			default:
				fmt.Printf("  FAIL  %s\n", r.Interaction)
				for _, f := range r.Failures {
					if f.Severity == verify.SeverityWarning {
						fmt.Printf("        [warning] %s\n", f)
					} else {
						fmt.Printf("        %s\n", f)
					}
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
