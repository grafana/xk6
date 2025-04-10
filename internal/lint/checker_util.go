package lint

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/grafana/k6foundry"
)

func build(ctx context.Context, module string, dir string) (string, error) {
	exe, err := os.CreateTemp("", "k6-*.exe")
	if err != nil {
		return "", err
	}

	const exePerm = 0o700

	if err = os.Chmod(exe.Name(), exePerm); err != nil {
		return "", err
	}

	var out bytes.Buffer

	var result error

	defer func() {
		if result != nil {
			_, _ = io.Copy(os.Stderr, &out)
			fmt.Fprintln(os.Stderr)
		}
	}()

	foundry, err := k6foundry.NewNativeFoundry(
		ctx,
		k6foundry.NativeFoundryOpts{
			Logger: slog.New(slog.NewTextHandler(&out, &slog.HandlerOptions{Level: slog.LevelError})),
			Stdout: &out,
			Stderr: &out,
			GoOpts: k6foundry.GoOpts{
				CopyGoEnv: true,
				Env:       map[string]string{"GOWORK": "off"},
			},
		},
	)
	if err != nil {
		result = err

		return "", result
	}

	platform, err := k6foundry.NewPlatform(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		result = err

		return "", result
	}

	_, result = foundry.Build(ctx, platform, "latest", []k6foundry.Module{{Path: module, ReplacePath: dir}}, nil, nil, exe)

	if result != nil {
		return "", result
	}

	if err = exe.Close(); err != nil {
		return "", err
	}

	return exe.Name(), nil
}

func findFile(rex *regexp.Regexp, dirs ...string) (string, string, error) {
	for idx, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if idx == 0 {
				return "", "", err
			}

			continue
		}

		script := ""

		for _, entry := range entries {
			if rex.MatchString(entry.Name()) {
				script = entry.Name()

				break
			}
		}

		if len(script) > 0 {
			return filepath.Join(dir, script), script, nil
		}
	}

	return "", "", nil
}
