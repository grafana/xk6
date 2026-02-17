package cmd

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed help/x.md
var xHelp string

func xCmd() *cobra.Command {
	opts := newBuildOptions()

	cmd := &cobra.Command{
		Use:   "x [flags] [--] [k6-flags] [subcommand] [subcommand-flags]",
		Short: shortHelp(xHelp),
		Long:  xHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runK6Command(cmd.Context(), opts, "x", args)
		},
		DisableAutoGenTag: true,
	}

	flags := cmd.Flags()

	flags.SortFlags = false

	cobra.CheckErr(buildCommonFlags(flags, opts))

	return cmd
}
