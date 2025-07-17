package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"

	"github.com/Masterminds/semver/v3"
	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"github.com/spf13/cobra"
	"go.k6.io/xk6/internal/sync"
)

//go:embed help/sync.md
var syncHelp string

type syncOptions struct {
	k6version string
	dryRun    bool
	out       string
	compact   bool
	json      bool
	markdown  bool
	quiet     bool
}

func syncCmd() *cobra.Command {
	opts := new(syncOptions)

	cmd := &cobra.Command{
		Use:   "sync [flags]",
		Short: shortHelp(syncHelp),
		Long:  syncHelp,
		Args:  cobra.NoArgs,
		PreRun: func(cmd *cobra.Command, _ []string) {
			opts.json = opts.json || opts.compact
			opts.quiet = cmd.Flags().Lookup("quiet").Changed
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return syncRunE(cmd.Context(), opts)
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
	flags.StringVarP(&opts.out, "out", "o", "",
		"Write output to file instead of stdout")
	flags.BoolVar(&opts.json, "json", false,
		"Generate JSON output")
	flags.BoolVarP(&opts.compact, "compact", "c", false,
		"Compact instead of pretty-printed JSON output")
	flags.BoolVarP(&opts.markdown, "markdown", "m", false,
		"Generate Markdown output")

	return cmd
}

func syncRunE(ctx context.Context, opts *syncOptions) (problem error) {
	result, err := sync.Sync(ctx, ".", &sync.Options{
		DryRun:    opts.dryRun,
		K6Version: opts.k6version,
	})
	if err != nil {
		return err
	}

	output := colorable.NewColorableStdout()

	if len(opts.out) > 0 {
		file, err := os.Create(opts.out)
		if err != nil {
			return err
		}

		defer func() {
			err := file.Close()
			if problem == nil && err != nil {
				problem = err
			}
		}()

		output = file
	}

	if opts.quiet {
		return nil
	}

	if opts.json {
		return jsonOutput(result, output, opts.compact)
	}

	if opts.markdown {
		return markdownSyncOutput(result, output)
	}

	textSyncOutput(result, output)

	return nil
}

func textSyncOutput(result *sync.Result, output io.Writer) {
	bold := color.New(color.FgHiWhite, color.Bold).SprintfFunc()

	downgrade := color.New(color.FgYellow).FprintfFunc()
	upgrade := color.New(color.FgGreen).FprintfFunc()
	plain := color.New(color.FgWhite).FprintfFunc()

	plain(output, "Dependencies have been synchronized with k6 %s.\n\n", bold(result.K6Version))

	plain(output, "Changes\n───────\n")

	for _, change := range result.Changes {
		fprintf := downgrade
		symbol := "▼"

		if isUpgrade(change) {
			fprintf = upgrade
			symbol = "▲"
		}

		fprintf(output, "%s %s\n", symbol, change.Module)
		plain(output, "  %s => %s\n", change.From, change.To)
	}

	plain(output, "\n")
}

func isUpgrade(change *sync.Change) bool {
	to, err := semver.NewVersion(change.To)
	if err != nil {
		return false
	}

	from, err := semver.NewVersion(change.From)
	if err != nil {
		return false
	}

	return to.GreaterThan(from)
}

func markdownSyncOutput(result *sync.Result, output io.Writer) error {
	_, err := fmt.Fprintf(output, "Dependencies have been synchronized with k6 `%s`.\n\n", result.K6Version)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(output, "**Changes**\n\n")
	if err != nil {
		return err
	}

	for _, change := range result.Changes {
		_, err = fmt.Fprintf(output, "- %s\n  `%s` => `%s`\n", change.Module, change.From, change.To)
		if err != nil {
			return err
		}
	}

	return nil
}
