// Copyright 2020 Matthew Holt
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"go.k6.io/xk6"
)

var (
	errExpectedValue  = errors.New("expected value")
	errInvalidValue   = errors.New("invalid value")
	errMissingFlag    = errors.New("missing flag")
	errMissingReplace = errors.New("missing replace")
)

type buildOps struct {
	K6Version      string
	Extensions     []string
	Replacements   []string
	OutFile        string
	OutputOverride bool
}

func main() {
	log := slog.Default()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go trapSignals(ctx, log, cancel)

	if len(os.Args) > 1 && os.Args[1] == "build" {
		if err := runBuild(ctx, log, os.Args[2:]); err != nil {
			log.Error(fmt.Sprintf("build error %v", err))
			os.Exit(1)
		}

		return
	}

	if err := runDev(ctx, log, os.Args[1:]); err != nil {
		log.Error(fmt.Sprintf("run error %v", err))
		os.Exit(1)
	}
}

func runBuild(ctx context.Context, log *slog.Logger, args []string) error {
	opts, err := parseBuildOpts(args)
	if err != nil {
		return fmt.Errorf("parsing options %w", err)
	}

	builder := xk6.FromOSEnv()
	if opts.K6Version != "" {
		builder.K6Version = opts.K6Version
	}

	builder.Extensions = opts.Extensions
	builder.Replacements = opts.Replacements

	// perform the build
	if err := builder.Build(ctx, log, opts.OutFile); err != nil {
		return err
	}

	// prove the build is working by printing the version
	output := opts.OutFile
	if runtime.GOOS == os.Getenv("GOOS") && runtime.GOARCH == os.Getenv("GOARCH") {
		if !filepath.IsAbs(output) {
			output = "." + string(filepath.Separator) + output
		}

		_, _ = fmt.Fprintf(os.Stdout, "\n%s version\n", output)
		cmd := exec.Command(output, "version") // #nosec G204
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("executing k6 %w", err)
		}
	}

	if !opts.OutputOverride {
		path, _ := os.Getwd()
		_, _ = fmt.Fprintln(
			os.Stdout,
			"\nxk6 has now produced a new k6 binary which may be different than the command on your system path!",
		)
		_, _ = fmt.Fprintf(os.Stdout, "Be sure to run '%v run <SCRIPT_NAME>' from the '%v' directory.\n", output, path)
	}

	return nil
}

func runDev(ctx context.Context, log *slog.Logger, args []string) error {
	// get current/main module name
	cmd := exec.Command("go", "list", "-m")
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("exec %v: %w: %s", cmd.Args, err, string(out))
	}

	currentModule := strings.TrimSpace(string(out))

	// get the root directory of the main module
	cmd = exec.Command("go", "list", "-m", "-f={{.Dir}}")
	cmd.Stderr = os.Stderr

	out, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("exec %v: %w: %s", cmd.Args, err, string(out))
	}

	moduleDir := strings.TrimSpace(string(out))

	// make sure the module being developed is replaced
	// so that the local copy is used
	replacements := []string{
		fmt.Sprintf("%s=%s", currentModule, moduleDir),
	}

	// replace directives only apply to the top-level/main go.mod,
	// and since this tool is a carry-through for the user's actual
	// go.mod, we need to transfer their replace directives through
	// to the one we're making
	cmd = exec.Command("go", "list", "-mod=readonly", "-m", "-f={{if .Replace}}{{.Path}}={{.Replace}}{{end}}", "all")
	cmd.Stderr = os.Stderr

	out, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("exec %v: %w: %s", cmd.Args, err, string(out))
	}

	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Split(line, "=")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			continue
		}

		replacements = append(replacements, fmt.Sprintf("%s=%s", parts[0], parts[1]))
	}

	// reconcile remaining path segments; for example if a module foo/a
	// is rooted at directory path /home/foo/a, but the current directory
	// is /home/foo/a/b, then the package to import should be foo/a/b
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to determine current directory: %w", err)
	}

	importPath := normalizeImportPath(currentModule, cwd, moduleDir)

	// create a builder with options from environment variables
	builder := xk6.FromOSEnv()

	// set the current module as dependency
	builder.Extensions = []string{importPath}

	// update replacements
	builder.Replacements = replacements

	outfile := defaultK6OutputFile()

	err = builder.Build(ctx, log, outfile)
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("Running %v\n\n", append([]string{outfile}, args...)))
	cmd = exec.Command(outfile, args...) // #nosec G204
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Wait()
}

// It will be refactored soon.
func parseBuildOpts(args []string) (buildOps, error) { //nolint:cyclop,funlen
	opts := buildOps{
		OutFile: defaultK6OutputFile(),
	}

	var argK6Version string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--with":
			if i == len(args)-1 {
				return buildOps{}, fmt.Errorf("%w after --with flag", errExpectedValue)
			}

			i++

			_, err := validateModule(args[i])
			if err != nil {
				return buildOps{}, err
			}

			opts.Extensions = append(opts.Extensions, args[i])

		case "--replace":
			if i == len(args)-1 {
				return buildOps{}, fmt.Errorf("%w after --replace flag", errExpectedValue)
			}

			i++

			hasReplace, err := validateModule(args[i])
			if err != nil {
				return buildOps{}, err
			}

			if !hasReplace {
				return buildOps{}, errMissingReplace
			}

			opts.Replacements = append(opts.Replacements, args[i])

		case "--output":
			if i == len(args)-1 {
				return buildOps{}, fmt.Errorf("%w after --output flag", errExpectedValue)
			}

			i++
			opts.OutFile = args[i]
			opts.OutputOverride = true

		default:
			if argK6Version != "" {
				return buildOps{}, fmt.Errorf("%w k6 version already set at %s", errMissingFlag, argK6Version)
			}

			argK6Version = args[i]
		}
	}

	// prefer k6 version from command line argument over env var
	if argK6Version != "" {
		opts.K6Version = argK6Version
	}

	return opts, nil
}

func normalizeImportPath(currentModule, cwd, moduleDir string) string {
	return path.Join(currentModule, filepath.ToSlash(strings.TrimPrefix(cwd, moduleDir)))
}

func trapSignals(ctx context.Context, log *slog.Logger, cancel context.CancelFunc) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	select {
	case <-sig:
		log.Info("SIGINT: Shutting down")
		cancel()
	case <-ctx.Done():
		return
	}
}

func defaultK6OutputFile() string {
	if runtime.GOOS == "windows" {
		return ".\\k6.exe"
	}

	return "./k6"
}

// validateModule checks if the argument is a valid go module specification
// if the argument has a replacement, it also checks if the replacement is valid and return true.
func validateModule(arg string) (bool, error) {
	module, replace, replaceSep := strings.Cut(arg, "=")

	modulePath, modVersion, versionSep := strings.Cut(module, "@")
	if modulePath == "" {
		return false, fmt.Errorf("%w: missing module path %q", errInvalidValue, arg)
	}

	if versionSep && modVersion == "" {
		return false, fmt.Errorf("%w: missing module version %q", errInvalidValue, arg)
	}

	if !replaceSep {
		return false, nil
	}

	if replace == "" {
		return false, fmt.Errorf("%w: missing replacement %q", errExpectedValue, arg)
	}

	replacePath, replaceVersion, versionSep := strings.Cut(replace, "@")
	if replacePath == "" {
		return false, fmt.Errorf("%w: missing replacePath path %q", errExpectedValue, arg)
	}

	if versionSep && replaceVersion == "" {
		return false, fmt.Errorf("%w: missing module version %q", errExpectedValue, arg)
	}

	return true, nil
}
