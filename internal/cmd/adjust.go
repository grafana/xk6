package cmd

import (
	"context"
	_ "embed"
	"log/slog"

	"github.com/spf13/cobra"
	"go.k6.io/xk6/internal/scaffold"
)

//go:embed help/adjust.md
var adjustHelp string

type adjustOptions struct {
	directory string
	container bool
	scaffold.Sample
}

func adjustCmd() *cobra.Command {
	opts := new(adjustOptions)

	cmd := &cobra.Command{
		Use:   "adjust [flags] [directory]",
		Short: shortHelp(adjustHelp),
		Long:  adjustHelp,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.directory = args[0]
			} else {
				opts.directory = "."
			}

			return adjustRunE(cmd.Context(), opts)
		},
		DisableAutoGenTag: true,
	}

	cmd.SetContext(context.Background())

	flags := cmd.Flags()

	flags.SortFlags = false

	flags.StringVarP(&opts.Description, "description", "d", "", "A short, on-sentence description of the extension")
	flags.StringVarP(&opts.Package, "package", "p", "", "The go package name for the extension")
	flags.BoolVar(&opts.container, "dev-container", false, "Run in a development container")

	return cmd
}

func probeModuleFromGitURL(opts *adjustOptions) bool {
	if len(opts.Module) == 0 {
		gitURL, err := getGitURL(opts.directory)
		if err != nil {
			slog.Info("Missing git origin URL, skipping customization", "directory", opts.directory)

			return false
		}

		opts.Module, err = gitURLToModule(gitURL)
		if err != nil {
			slog.Info("Unsupported git URL, skipping customization", "url", gitURL)

			return false
		}
	}

	return true
}

func probeDescriptionFromGitURL(ctx context.Context, opts *adjustOptions) bool {
	if len(opts.Description) == 0 {
		gitURL, err := getGitURL(opts.directory)
		if err != nil {
			slog.Info("Missing git origin URL, skipping customization", "directory", opts.directory)

			return false
		}

		description, err := getDescription(ctx, gitURL)
		if err != nil {
			slog.Info("Failed to get description from git URL, skipping customization", "url", gitURL)

			return false
		}

		opts.Description = description
	}

	return true
}

func adjustRunE(ctx context.Context, opts *adjustOptions) error {
	modulePath, err := getModulePath(opts.directory)
	if err != nil {
		return err
	}

	ok, sample := scaffold.IsSample(modulePath)
	if !ok {
		// Exit silently if run in a development container
		if !opts.container {
			slog.Info("Not a sample extension, skipping customization", "module", modulePath)
		}

		return nil
	}

	if !probeModuleFromGitURL(opts) {
		return nil
	}

	if !probeDescriptionFromGitURL(ctx, opts) {
		return nil
	}

	if err := scaffold.Adjust(opts.directory, sample, &opts.Sample); err != nil {
		return err
	}

	slog.Info("Extension customized", "directory", opts.directory)
	slog.Info("Please review the changes and commit them to your repository")

	return nil
}
