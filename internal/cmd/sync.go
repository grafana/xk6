package cmd

import (
	"context"
	_ "embed"

	"github.com/spf13/cobra"
	"go.k6.io/xk6/internal/sync"
)

//go:embed help/sync.md
var syncHelp string

type syncOptions struct {
	k6version string
	dryRun    bool
}

func syncCmd() *cobra.Command {
	opts := new(syncOptions)

	cmd := &cobra.Command{
		Use:   "sync [flags]",
		Short: shortHelp(syncHelp),
		Long:  syncHelp,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return sync.Sync(cmd.Context(), ".", &sync.Options{
				DryRun:    opts.dryRun,
				K6Version: opts.k6version,
			})
		},
		DisableAutoGenTag: true,
	}

	cmd.SetContext(context.Background())

	flags := cmd.Flags()

	flags.SortFlags = false

	flags.StringVarP(&opts.k6version, "k6-version", "k", "",
		"The k6 version to use for synchronization (default from go.mod)")
	flags.BoolVarP(&opts.dryRun, "dry-run", "n", false,
		"Do not make any changes, only log them")

	return cmd
}
