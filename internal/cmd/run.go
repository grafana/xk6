package cmd

import (
	"context"
	_ "embed"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

//go:embed help/run.md
var runHelp string

func runCmd() *cobra.Command {
	opts := newBuildOptions()

	cmd := &cobra.Command{
		Use:   "run [flags] [--] [k6-flags] script",
		Short: shortHelp(runHelp),
		Long:  runHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRunE(cmd.Context(), opts, args)
		},
		DisableAutoGenTag: true,
	}

	flags := cmd.Flags()

	flags.SortFlags = false

	cobra.CheckErr(buildCommonFlags(flags, opts))

	return cmd
}

func runRunE(ctx context.Context, opts *buildOptions, args []string) error {
	cleanup, err := buildK6OnTheFly(ctx, opts)
	if err != nil {
		return err
	}

	defer cleanup()

	k6args := make([]string, len(args)+1)

	k6args[0] = "run"

	copy(k6args[1:], args)

	cmd := exec.CommandContext(ctx, opts.output, k6args...) // #nosec G204

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Wait()
}
