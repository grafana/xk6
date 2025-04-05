package cmd

import (
	"context"
	_ "embed"
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/grafana/k6foundry"
	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
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
	dir, err := os.MkdirTemp("", "xk6-run-*")
	if err != nil {
		return err
	}

	mfile, moddir, err := getModfile()
	if err != nil {
		return err
	}

	opts.extensions.modules = append(
		opts.extensions.modules,
		k6foundry.Module{Path: mfile.Module.Mod.Path, ReplacePath: moddir},
	)

	for _, rep := range mfile.Replace {
		opts.replacements.modules = append(
			opts.replacements.modules,
			k6foundry.Module{Path: rep.Old.Path, ReplacePath: rep.New.Path},
		)
	}

	opts.output = filepath.Join(dir, filepath.Base(defaultK6Output()))

	defer os.RemoveAll(dir) //nolint:errcheck

	_, err = buildK6(ctx, opts)
	if err != nil {
		return err
	}

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

func getModfile() (*modfile.File, string, error) {
	filename, err := findModfile()
	if err != nil {
		return nil, "", err
	}

	if len(filename) == 0 {
		return nil, "", errNoModfile
	}

	data, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, "", err
	}

	mf, err := modfile.Parse(filename, data, nil)
	if err != nil {
		return nil, "", err
	}

	return mf, filepath.Dir(filename), nil
}

func findModfile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for ; len(dir) != 1 || dir[0] != filepath.Separator; dir = filepath.Dir(dir) {
		filename := filepath.Join(dir, "go.mod")
		if info, err := os.Stat(filename); err == nil && !info.IsDir() {
			return filename, nil
		}
	}

	return "", nil
}

var errNoModfile = errors.New("go.mod file not found in current directory or any parent directory")
