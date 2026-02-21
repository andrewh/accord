// Verify subcommand: runs contract interactions against a live provider.
package cli

import (
	"fmt"
	"time"

	"github.com/andrewh/accord/internal/contract"
	"github.com/andrewh/accord/internal/lint"
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
	Example: `  accord verify --provider-url http://localhost:8080 contracts/user-api.yaml
  accord verify --provider-url http://localhost:8080 --timeout 60 contracts/*.yaml`,
	Args: cobra.MinimumNArgs(1),
	RunE: runVerify,
}

func runVerify(cmd *cobra.Command, args []string) error {
	if err := verify.ValidateProviderURL(providerURL); err != nil {
		cmd.SilenceUsage = true
		fmt.Fprintln(cmd.ErrOrStderr(), err) //nolint:errcheck
		return err
	}

	hasFailures := false
	linter := lint.New()

	for _, path := range args {
		result, err := contract.ParseFile(path)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", path, err) //nolint:errcheck
			hasFailures = true
			continue
		}

		out := cmd.OutOrStdout()
		diags := linter.Lint(result.Contract, result.Node)
		for _, d := range diags {
			fmt.Fprintln(out, lint.FormatDiagnostic(path, d)) //nolint:errcheck
		}
		if lint.HasErrors(diags) {
			hasFailures = true
			continue
		}

		c := result.Contract
		fmt.Fprintf(out, "Verifying %s (%s -> %s)\n", path, c.Consumer.Name, c.Provider.Name) //nolint:errcheck

		results := verify.Verify(c, providerURL, time.Duration(timeout)*time.Second)
		for _, r := range results {
			hasWarnings := len(r.Failures) > 0 && r.Passed
			switch {
			case r.Passed && !hasWarnings:
				fmt.Fprintf(out, "  PASS  %s\n", r.Interaction) //nolint:errcheck
			case r.Passed && hasWarnings:
				fmt.Fprintf(out, "  WARN  %s\n", r.Interaction) //nolint:errcheck
				for _, f := range r.Failures {
					fmt.Fprintf(out, "        [warning] %s\n", f) //nolint:errcheck
				}
			default:
				fmt.Fprintf(out, "  FAIL  %s\n", r.Interaction) //nolint:errcheck
				for _, f := range r.Failures {
					if f.Severity == verify.SeverityWarning {
						fmt.Fprintf(out, "        [warning] %s\n", f) //nolint:errcheck
					} else {
						fmt.Fprintf(out, "        %s\n", f) //nolint:errcheck
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
	fmt.Fprintln(cmd.OutOrStdout(), "\nAll interactions passed.") //nolint:errcheck
	return nil
}
