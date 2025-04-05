// Package docsme allows the documentation of command line tools to be automatically updated based on their CLI help.
// Markdown documentation can be generated from Cobra command definitions.
// The generated documentation can be used to update part of README.md or can be written to a separate file.
package docsme

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/spf13/cobra"
)

const (
	defaultRegion  = "cli"
	defaultHeading = 2

	standaloneName = "docsme"
	subcommandName = "docs"
)

var errRegionNotFound = errors.New("region not found")

// New is a factory function for creating the docsme Cobra subcommand.
// The advantage of using docsme as a subcommand is that documentation can be generated at any time with the CLI tool.
// Since docsme is a small library without additional dependencies, embedding it does not increase the size of the CLI.
// The name of the created Cobra command will be 'docs'.
func New() *cobra.Command {
	return newCommand(false)
}

// For is a factory function for creating docsme Cobra root command.
// The created root command can be used as a build-time tool.
// The advantage of using docsme as a build-time tool is that the docsme library is not built into the CLI at all.
func For(root *cobra.Command) *cobra.Command {
	cmd := newCommand(true)

	saveRoot(cmd, root)

	return cmd
}

type rootKey struct{}

func saveRoot(cmd *cobra.Command, root *cobra.Command) {
	cmd.SetContext(context.WithValue(context.Background(), rootKey{}, root))
}

func loadRoot(cmd *cobra.Command) *cobra.Command {
	if cmd.HasParent() {
		return cmd.Root()
	}

	ctx := cmd.Context()
	if ctx == nil {
		return cmd
	}

	val := ctx.Value(rootKey{})
	if reflect.ValueOf(val).IsZero() {
		return cmd
	}

	root, ok := val.(*cobra.Command)
	if !ok {
		return cmd
	}

	return root
}

type options struct {
	heading int
	output  string
	force   bool
	region  string
}

//go:embed docsme.md
var help string

func newCommand(standalone bool) *cobra.Command {
	opts := &options{
		heading: defaultHeading,
		region:  defaultRegion,
	}

	name := subcommandName
	if standalone {
		name = standaloneName
	}

	cmd := &cobra.Command{
		Use:   name,
		Short: "Keep documentation up to date based on CLI help",
		Long:  help,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(loadRoot(cmd), opts, cmd.OutOrStdout())
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	flags := cmd.Flags()

	flags.SortFlags = false

	flags.StringVarP(&opts.output, "output", "o", opts.output, "Output filename (default stdout)")
	flags.StringVarP(&opts.region, "region", "r", opts.region, "File region to update")
	flags.BoolVarP(&opts.force, "force", "f", opts.force, "Force file overwrite")
	flags.IntVar(&opts.heading, "heading", opts.heading, "Initial heading level")

	return cmd
}

// run generates the markdown documentation recursively based on cobra Command.
func run(root *cobra.Command, opts *options, out io.Writer) error {
	var buff bytes.Buffer

	const (
		maxHeadingOffset = 3
		permFile         = 0o600
	)

	gen := &mdgen{min(max(0, opts.heading-1), maxHeadingOffset)}

	if err := gen.generate(root, &buff); err != nil {
		return err
	}

	if len(opts.output) == 0 {
		_, err := io.Copy(out, &buff)

		return err
	}

	filename := filepath.Clean(opts.output)

	src, err := os.ReadFile(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return os.WriteFile(filename, buff.Bytes(), permFile)
		}

		return err
	}

	res, found, err := replace(src, opts.region, buff.Bytes())
	if err != nil {
		return err
	}

	if found {
		src = res
	} else if !opts.force {
		return fmt.Errorf("%w: %s", errRegionNotFound, opts.region)
	}

	return os.WriteFile(filename, src, permFile)
}
