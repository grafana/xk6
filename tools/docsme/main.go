// Package main contains a build-time documentation generation tool.
// This tool generates the README.md file based on the CLI help.
package main

import (
	"github.com/spf13/cobra"
	"github.com/szkiba/docsme"
	"go.k6.io/xk6/internal/cmd"
)

func main() {
	cobra.EnableCommandSorting = false

	cobra.CheckErr(docsme.For(cmd.New(nil)).Execute())
}
