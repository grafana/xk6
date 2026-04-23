package cmd

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

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
			if !opts.outputChanged && opts.os == "windows" && !strings.HasSuffix(opts.output, ".exe") {
				opts.output += ".exe"
			}

			return buildRunE(cmd.Context(), cmd.OutOrStdout(), opts)
		},
		DisableAutoGenTag: true,
	}

	flags := cmd.Flags()

	flags.SortFlags = false

	flags.StringVarP(&opts.output, "output", "o", defaultK6Output(), "Output filename")

	cobra.CheckErr(buildCommonFlags(flags, opts))

	return cmd
}

func buildRunE(ctx context.Context, stdout io.Writer, opts *buildOptions) error {
	info, err := buildK6(ctx, opts)
	if err != nil {
		return err
	}

	slog.Info("Successful build", "platform", info.Platform)

	for _, w := range info.Warnings {
		slog.Warn(w.Message)
	}

	k6modPath := info.K6ModPath
	k6ver := info.ModVersions[k6modPath]
	if k6modPath != "" {
		delete(info.ModVersions, k6modPath)
		slog.Info("added", "module", k6modPath, "version", k6ver)
	}

	for name, version := range info.ModVersions {
		slog.Info("added", "module", name, "version", version)
	}

	if k6modPath != "" {
		slog.Info("A new binary has been built based on k6", "version", k6ver)
	}

	k6latest, err := sync.GetLatestVersion(ctx, k6modPath)
	if err == nil && k6ver != k6latest {
		slog.Warn("Newer k6 version available", "actual", k6ver, "latest", k6latest)
	} else if err != nil {
		slog.Warn("Failed to get latest k6 version", "error", err)
	}

	if !opts.outputChanged {
		buildCompatMessage(stdout, opts.output)
	}

	return nil
}

const buildCompatMessageFmt = `
xk6 has now produced a new k6 binary which may be different than the command on your system path!
Be sure to run '%v run <SCRIPT_NAME>' from the '%v' directory.
`

func buildCompatMessage(stdout io.Writer, exe string) {
	abs, _ := filepath.Abs(exe)

	_, _ = fmt.Fprintf(stdout, buildCompatMessageFmt, exe, filepath.Dir(abs))
}
