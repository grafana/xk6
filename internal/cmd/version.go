package cmd

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed help/version.md
var versionHelp string

func versionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: shortHelp(versionHelp),
		Long:  versionHelp,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := cmd.Root()
			root.SetArgs([]string{"--version"})

			return root.Execute()
		},
		DisableAutoGenTag: true,
	}

	return cmd
}
