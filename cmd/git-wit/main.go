// Package main provides the git-wit CLI entrypoint.
package main

import (
	"context"
	"os"

	"github.com/Warashi/git-wit/internal/cli"
)

func main() {
	err := cli.NewRootCommand().ExecuteContext(context.Background())
	if err != nil {
		os.Exit(1)
	}
}
