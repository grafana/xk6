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

type BuildOps struct {
	K6Version      string
	Extensions     []xk6.Dependency
	Replacements   []xk6.Replace
	OutFile        string
	OutputOverride bool
}

func main() {
	log := slog.New(
		slog.NewTextHandler(
			os.Stderr,
			&slog.HandlerOptions{
				Level: slog.LevelDebug,
			},
		),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go trapSignals(ctx, log, cancel)

	if len(os.Args) > 1 && os.Args[1] == "build" {
		if err := runBuild(ctx, log, os.Args[2:]); err != nil {
			log.Error(fmt.Sprintf("build error %v", err))
		}
		return
	}

	if err := runDev(ctx, log, os.Args[1:]); err != nil {
		log.Error(fmt.Sprintf("run error %v", err))
	}
}

func runBuild(ctx context.Context, log *slog.Logger, args []string) error {
	opts, err := parseBuildOpts(args)
	if err != nil {
		return fmt.Errorf("parsing options %v", err)
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
		fmt.Println()
		fmt.Printf("%s version\n", output)
		cmd := exec.Command(output, "version")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("executing k6 %v", err)
		}
	}

	if !opts.OutputOverride {
		path, _ := os.Getwd()
		fmt.Println()
		fmt.Println("xk6 has now produced a new k6 binary which may be different than the command on your system path!")
		fmt.Printf("Be sure to run '%v run <SCRIPT_NAME>' from the '%v' directory.\n", output, path)
	}

	return nil
}

func runDev(ctx context.Context, log *slog.Logger, args []string) error {
	// get current/main module name
	cmd := exec.Command("go", "list", "-m")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("exec %v: %v: %s", cmd.Args, err, string(out))
	}
	currentModule := strings.TrimSpace(string(out))

	// get the root directory of the main module
	cmd = exec.Command("go", "list", "-m", "-f={{.Dir}}")
	cmd.Stderr = os.Stderr
	out, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("exec %v: %v: %s", cmd.Args, err, string(out))
	}
	moduleDir := strings.TrimSpace(string(out))

	// make sure the module being developed is replaced
	// so that the local copy is used
	replacements := []xk6.Replace{
		xk6.NewReplace(currentModule, moduleDir),
	}

	// replace directives only apply to the top-level/main go.mod,
	// and since this tool is a carry-through for the user's actual
	// go.mod, we need to transfer their replace directives through
	// to the one we're making
	cmd = exec.Command("go", "list", "-mod=readonly", "-m", "-f={{if .Replace}}{{.Path}} => {{.Replace}}{{end}}", "all")
	cmd.Stderr = os.Stderr
	out, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("exec %v: %v: %s", cmd.Args, err, string(out))
	}
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Split(line, "=>")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		replacements = append(replacements, xk6.NewReplace(
			strings.TrimSpace(parts[0]),
			strings.TrimSpace(parts[1]),
		))
	}

	// reconcile remaining path segments; for example if a module foo/a
	// is rooted at directory path /home/foo/a, but the current directory
	// is /home/foo/a/b, then the package to import should be foo/a/b
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to determine current directory: %v", err)
	}
	importPath := normalizeImportPath(currentModule, cwd, moduleDir)

	// create a builder with options from environment variables
	builder := xk6.FromOSEnv()

	// set the current module as dependency
	builder.Extensions = []xk6.Dependency{
		{PackagePath: importPath},
	}
	// update replacements
	builder.Replacements = replacements

	outfile := defaultK6OutputFile()
	err = builder.Build(ctx, log, outfile)
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("Running %v\n\n", append([]string{outfile}, args...)))
	cmd = exec.Command(outfile, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Wait()
}

func parseBuildOpts(args []string) (BuildOps, error) {
	opts := BuildOps{
		OutFile: defaultK6OutputFile(),
	}

	var argK6Version string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--with":
			if i == len(args)-1 {
				return BuildOps{}, fmt.Errorf("expected value after --with flag")
			}
			i++
			mod, ver, repl, err := splitWith(args[i])
			if err != nil {
				return BuildOps{}, err
			}
			mod = strings.TrimSuffix(mod, "/") // easy to accidentally leave a trailing slash if pasting from a URL, but is invalid for Go modules
			opts.Extensions = append(opts.Extensions, xk6.Dependency{
				PackagePath: mod,
				Version:     ver,
			})
			if repl != "" {
				repl, err = expandPath(repl)
				if err != nil {
					return BuildOps{}, err
				}
				opts.Replacements = append(opts.Replacements, xk6.NewReplace(mod, repl))
			}

		case "--replace":
			if i == len(args)-1 {
				return BuildOps{}, fmt.Errorf("expected value after --replace flag")
			}
			i++
			mod, _, repl, err := splitWith(args[i])
			if err != nil {
				return BuildOps{}, err
			}
			if repl == "" {
				return BuildOps{}, fmt.Errorf("replace value must be of format 'module=replace' or 'module=replace@version'")
			}
			// easy to accidentally leave a trailing slash if pasting from a URL, but is invalid for Go modules
			mod = strings.TrimSuffix(mod, "/")
			repl, err = expandPath(repl)
			if err != nil {
				return BuildOps{}, err
			}
			opts.Replacements = append(opts.Replacements, xk6.NewReplace(mod, repl))

		case "--output":
			if i == len(args)-1 {
				return BuildOps{}, fmt.Errorf("expected value after --output flag")
			}
			i++
			opts.OutFile = args[i]
			opts.OutputOverride = true

		default:
			if argK6Version != "" {
				return BuildOps{}, fmt.Errorf("missing flag; k6 version already set at %s", argK6Version)
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

func expandPath(path string) (string, error) {
	// expand local directory
	if path == "." {
		if cwd, err := os.Getwd(); err != nil {
			return "", err
		} else {
			return cwd, nil
		}
	}
	// expand ~ as shortcut for home directory
	if strings.HasPrefix(path, "~") {
		if home, err := os.UserHomeDir(); err != nil {
			return "", err
		} else {
			return strings.Replace(path, "~", home, 1), nil
		}
	}
	return path, nil
}

func splitWith(arg string) (module, version, replace string, err error) {
	const versionSplit, replaceSplit = "@", "="

	parts := strings.SplitN(arg, replaceSplit, 2)
	if len(parts) > 1 {
		replace = parts[1]
	} else {
		replace = ""
	}

	module = parts[0]

	moduleParts := strings.SplitN(module, versionSplit, 2)
	if len(moduleParts) > 1 {
		module = moduleParts[0]
		version = moduleParts[1]
	}

	if module == "" {
		err = fmt.Errorf("module name is required")
	}

	return
}

func defaultK6OutputFile() string {
	if runtime.GOOS == "windows" {
		return ".\\k6.exe"
	}
	return "./k6"
}
