package cmd

import (
	"context"
	_ "embed"
	"log/slog"
	"path"

	"github.com/spf13/cobra"
	"go.k6.io/xk6/internal/scaffold"
)

//go:embed help/new.md
var newHelp string

type newOptions struct {
	kind   string
	parent string

	scaffold.Sample
}

func newCmd() *cobra.Command {
	opts := new(newOptions)

	cmd := &cobra.Command{
		Use:   "new [flags] module",
		Short: shortHelp(newHelp),
		Long:  newHelp,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Module = args[0]

			return newRunE(cmd.Context(), opts)
		},
		DisableAutoGenTag: true,
	}

	cmd.SetContext(context.Background())

	flags := cmd.Flags()

	flags.SortFlags = false

	flags.StringVarP(&opts.kind, "type", "t", "", "The type of template to use (javascript or output)")
	flags.StringVarP(&opts.Description, "description", "d", "", "A short, on-sentence description of the extension")
	flags.StringVarP(&opts.Package, "package", "p", "", "The go package name for the extension")
	flags.StringVarP(&opts.parent, "parent-dir", "C", ".", "The parent directory")

	return cmd
}

func newRunE(ctx context.Context, opts *newOptions) error {
	sam, err := scaffold.LookupSample(opts.Module, opts.kind)
	if err != nil {
		return err
	}

	if len(opts.Description) == 0 {
		opts.Description, _ = getDescription(ctx, moduleToGitURL(opts.Module))
	}

	if err := scaffold.Scaffold(ctx, sam, &opts.Sample, opts.parent); err != nil {
		return err
	}

	dir := path.Base(opts.Module)
	if opts.parent != "." {
		dir = path.Join(opts.parent, dir)
	}

	slog.Info("Extension created", "module", opts.Module, "directory", dir)

	return nil
}
