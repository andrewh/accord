// Accord CLI entrypoint. Parses arguments and dispatches to subcommands.
package main

import (
	"os"

	"github.com/andrewh/accord/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
