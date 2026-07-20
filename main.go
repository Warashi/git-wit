// Package main provides the git-wit CLI entrypoint.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Warashi/git-wit/internal/cli"
)

func main() {
	err := cli.NewRootCommand().ExecuteContext(context.Background())
	if err != nil {
		// The root command sets SilenceErrors, so nothing else reports err.
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
