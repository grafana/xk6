package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/grafana/k6foundry"
	"golang.org/x/mod/modfile"
)

func runK6Command(ctx context.Context, opts *buildOptions, k6cmd string, args []string) error {
	cleanup, err := buildK6OnTheFly(ctx, opts)
	if err != nil {
		return err
	}

	defer cleanup()

	k6args := make([]string, len(args)+1)

	k6args[0] = k6cmd

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

func buildK6OnTheFly(ctx context.Context, opts *buildOptions) (func(), error) {
	dir, err := os.MkdirTemp("", "xk6-build-*")
	if err != nil {
		return nil, err
	}

	mfile, moddir, err := getModfile()
	if err != nil {
		return nil, err
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

	_, err = buildK6(ctx, opts)
	if err != nil {
		return nil, err
	}

	return func() {
		_ = os.RemoveAll(dir)
	}, nil
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

		info, err := os.Stat(filename)
		if err == nil && !info.IsDir() {
			return filename, nil
		}
	}

	return "", nil
}

var errNoModfile = errors.New("go.mod file not found in current directory or any parent directory")
