package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/grafana/k6foundry"
	"github.com/spf13/pflag"
	"github.com/szkiba/efa"
)

type buildOptions struct {
	output       string
	extensions   *modules
	replacements *modules
	k6version    string
	k6repo       string
	os           string
	arch         string
	arm          string
	skipCleanup  int
	raceDetector int
	cgo          int
	buildFlags   []string

	outputChanged bool
}

func newBuildOptions() *buildOptions {
	opts := new(buildOptions)

	opts.extensions = new(modules)
	opts.replacements = &modules{replace: true}

	return opts
}

const (
	defaultK6Version    = "latest"
	defaultK6Repo       = "go.k6.io/k6"
	defaultCgo          = 0
	defaultRaceDetector = 0
	defaultSkipCleanup  = 0
	defaultBuildFlags   = "-trimpath,-ldflags=-s -w"
)

var nonGoEnvToCopy = []string{ //nolint:gochecknoglobals
	"HTTP_PROXY",
	"HTTPS_PROXY",
	"NO_PROXY",
}

func defaultK6Output() string {
	if runtime.GOOS == "windows" {
		return ".\\k6.exe"
	}

	return "./k6"
}

func buildCommonFlags(flags *pflag.FlagSet, opts *buildOptions) error {
	flags.Var(opts.extensions, "with", "Add one or more k6 extensions with Go module path")
	flags.Var(opts.replacements, "replace", "Replace one or more Go modules")
	flags.StringVarP(&opts.k6version, "k6-version", "k", defaultK6Version, "The k6 version to use for build")
	flags.StringVar(&opts.k6repo, "k6-repo", defaultK6Repo, "The k6 repository to use for the build")
	flags.StringVar(&opts.os, "os", runtime.GOOS, "The target operating system")
	flags.StringVar(&opts.arch, "arch", runtime.GOARCH, "The target architecture")
	flags.StringVar(&opts.arm, "arm", "", "The target ARM version")
	flags.IntVar(&opts.skipCleanup, "skip-cleanup", defaultSkipCleanup, "Keep the temporary build directory")
	flags.IntVar(&opts.raceDetector, "race-detector", defaultRaceDetector, "Enable/disable race detector")
	flags.IntVar(&opts.cgo, "cgo", defaultCgo, "Enable/disable cgo")
	flags.StringArrayVar(&opts.buildFlags, "build-flags", strings.Split(defaultBuildFlags, ","), "Specify Go build flags")

	flags.Lookup("cgo").NoOptDefVal = "1"
	flags.Lookup("skip-cleanup").NoOptDefVal = "1"
	flags.Lookup("race-detector").NoOptDefVal = "1"

	env := efa.New(flags, appname, nil)

	err := env.Bind("k6-repo", "build-flags", "race-detector", "skip-cleanup")
	if err != nil {
		return err
	}

	err = env.BindTo("os", "GOOS", "arch", "GOARCH", "arm", "GOARM")
	if err != nil {
		return err
	}

	env = efa.New(flags, "", nil)

	err = env.Bind("k6-version")
	if err != nil {
		return err
	}

	return env.BindTo("cgo", "CGO_ENABLED")
}

// copyNonGoEnv copies non-Go environment variables that might be needed for the build.
func copyNonGoEnv(env map[string]string) {
	for _, key := range nonGoEnvToCopy {
		if val, ok := os.LookupEnv(key); ok {
			env[key] = val
		}
	}
}

func newFoundry(ctx context.Context, opts *buildOptions) (k6foundry.Foundry, error) { //nolint:ireturn
	logger := slog.Default()

	env := make(map[string]string)

	// ANCHOR workaround only, ARM version should be supported by k6foundry
	if len(opts.arm) > 0 {
		env["GOARM"] = opts.arm
	}

	// ANCHOR workaround only, cgo flag should be supported by k6foundry
	env["CGO_ENABLED"] = strconv.Itoa(opts.cgo)

	// copy non-Go environment variables that might be needed for the build
	copyNonGoEnv(env)

	if opts.raceDetector != 0 {
		opts.buildFlags = append(opts.buildFlags, "-race")
	}

	if opts.cgo == 0 {
		raceDetector := false

		for _, f := range opts.buildFlags {
			raceDetector = raceDetector || strings.Contains(f, "-race")
		}

		if raceDetector && opts.cgo == 0 {
			slog.Warn("Enabling cgo because it is required by the race detector")

			env["CGO_ENABLED"] = "1"
		}
	}

	fopts := k6foundry.NativeFoundryOpts{
		GoOpts: k6foundry.GoOpts{
			CopyGoEnv: true,
			Env:       env,
		},
		SkipCleanup: opts.skipCleanup != 0,
		Logger:      logger,
	}

	if opts.k6repo != defaultK6Repo { // only set if different from default: k6foundry workaround
		fopts.K6Repo = opts.k6repo
	}

	if logger.Enabled(ctx, slog.LevelDebug) {
		fopts.Stdout = os.Stdout
		fopts.Stderr = os.Stderr
	}

	return k6foundry.NewNativeFoundry(ctx, fopts)
}

func buildK6(ctx context.Context, opts *buildOptions) (*k6foundry.BuildInfo, error) {
	foundry, err := newFoundry(ctx, opts)
	if err != nil {
		return nil, err
	}

	const outFilePerm = 0o777

	out, err := os.OpenFile(opts.output, os.O_WRONLY|os.O_CREATE, outFilePerm) // #nosec G302 G304
	if err != nil {
		return nil, err
	}

	// ANCHOR missing ARM version support in k6foundry
	platform, err := k6foundry.NewPlatform(opts.os, opts.arch)
	if err != nil {
		return nil, err
	}

	info, err := foundry.Build(
		ctx,
		platform,
		opts.k6version,
		opts.extensions.modules,
		opts.replacements.modules,
		opts.buildFlags,
		out,
	)
	if err != nil {
		_ = out.Close()

		return nil, err
	}

	if err := out.Close(); err != nil {
		return nil, err
	}

	return info, nil
}

type modules struct {
	replace bool
	modules []k6foundry.Module
}

func (m *modules) String() string {
	var buff strings.Builder

	for idx := range m.modules {
		if idx != 0 {
			buff.WriteRune(',')
		}

		buff.WriteString(m.modules[idx].String())
	}

	return buff.String()
}

func (m *modules) Set(val string) error {
	mod, err := k6foundry.ParseModule(val)
	if err != nil {
		return err
	}

	if m.replace {
		if len(mod.ReplacePath) == 0 {
			return fmt.Errorf("%w: missing replace", k6foundry.ErrInvalidDependencyFormat)
		}

		if len(mod.Version) > 0 {
			return fmt.Errorf("%w: module version is not allowed", k6foundry.ErrInvalidDependencyFormat)
		}
	}

	m.modules = append(m.modules, mod)

	return nil
}

func (m *modules) Type() string {
	if m.replace {
		return "module=replacement"
	}

	return "module[@version][=replacement]"
}
