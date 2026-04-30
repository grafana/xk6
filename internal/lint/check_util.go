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

	"go.k6.io/xk6/internal/sync"
)

func findFile(rex *regexp.Regexp, dirs ...string) (string, string, error) {
	for idx, dir := range dirs {
		entries, err := os.ReadDir(dir) //nolint:forbidigo
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

func build(ctx context.Context, module string, dir string) (string, error) {
	exe, err := os.CreateTemp("", "k6-*.exe") //nolint:forbidigo
	if err != nil {
		return "", err
	}

	const exePerm = 0o700

	err = os.Chmod(exe.Name(), exePerm) /* #nosec G302 G703 */ //nolint:forbidigo
	if err != nil {
		return "", err
	}

	var out bytes.Buffer

	var result error

	defer func() {
		if result != nil {
			_, _ = io.Copy(os.Stderr, &out) //nolint:forbidigo
			fmt.Fprintln(os.Stderr)         //nolint:forbidigo
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
	_, version, err := sync.ResolveK6ModuleForExtensions(ctx, []sync.ExtensionModule{
		{
			Path:      module,
			LocalPath: dir,
		},
	})
	if err != nil {
		result = err
		return "", err
	}

	_, result = foundry.Build(ctx, platform, version, []k6foundry.Module{{Path: module, ReplacePath: dir}}, nil, nil, exe)
	if result != nil {
		return "", result
	}

	err = exe.Close()
	if err != nil {
		return "", err
	}

	return exe.Name(), nil
}
