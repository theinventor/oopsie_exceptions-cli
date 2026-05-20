// Command oopsie is the JSON-first CLI for Oopsie exception tracking.
package main

import (
	"fmt"
	"os"

	"github.com/theinventor/oopsie_exceptions-cli/cmd"
	"github.com/theinventor/oopsie_exceptions-cli/internal/exitcode"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "oopsie:", err)
		os.Exit(exitcode.ExitCodeFor(err))
	}
}
