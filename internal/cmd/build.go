package cmd

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.k6.io/xk6/internal/sync"
)

//go:embed help/build.md
var buildHelp string

var errArgumentAndFlag = errors.New("k6 version was specified with both flag and argument")

func buildCmd() *cobra.Command {
	opts := newBuildOptions()

	cmd := &cobra.Command{
		Use:   "build [flags] [k6-version]",
		Short: shortHelp(buildHelp),
		Long:  buildHelp,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			k6v := cmd.Flags().Lookup("k6-version")

			if k6v.Changed && len(args) != 0 {
				return errArgumentAndFlag
			}

			if len(args) > 0 {
				opts.k6version = args[0]
			}

			opts.outputChanged = cmd.Flags().Lookup("output").Changed

			return buildRunE(cmd.Context(), opts)
		},
		DisableAutoGenTag: true,
	}

	flags := cmd.Flags()

	flags.SortFlags = false

	flags.StringVarP(&opts.output, "output", "o", defaultK6Output(), "Output filename")

	cobra.CheckErr(buildCommonFlags(flags, opts))

	return cmd
}

func buildRunE(ctx context.Context, opts *buildOptions) error {
	info, err := buildK6(ctx, opts)
	if err != nil {
		return err
	}

	slog.Info("Successful build", "platform", info.Platform)

	k6ver, ok := info.ModVersions[opts.k6repo]
	if ok {
		delete(info.ModVersions, opts.k6repo)
		slog.Info("added", "module", opts.k6repo, "version", k6ver)
	}

	for name, version := range info.ModVersions {
		slog.Info("added", "module", name, "version", version)
	}

	if ok {
		slog.Info("A new binary has been built based on k6", "version", k6ver)
	}

	k6latest, err := sync.GetLatestK6Version(ctx)
	if err == nil && k6ver != k6latest {
		slog.Warn("Newer k6 version available", "actual", k6ver, "latest", k6latest)
	} else if err != nil {
		slog.Warn("Failed to get latest k6 version", "error", err)
	}

	if !opts.outputChanged {
		buildCompatMessage(opts.output)
	}

	return nil
}

const buildCompatMessageFmt = `
xk6 has now produced a new k6 binary which may be different than the command on your system path!
Be sure to run '%v run <SCRIPT_NAME>' from the '%v' directory.
`

func buildCompatMessage(exe string) {
	abs, _ := filepath.Abs(exe)

	_, _ = fmt.Fprintf(os.Stdout, buildCompatMessageFmt, exe, filepath.Dir(abs))
}
