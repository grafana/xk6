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
	"go.k6.io/xk6/internal/sync"
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
	"HTTP_PROXY",          // required by git over http(s)
	"HTTPS_PROXY",         // required by git over http(s)
	"NO_PROXY",            // required by git over http(s)
	"SSH_AUTH_SOCK",       // required by git over ssh
	"SSH_AGENT_PID",       // required by git over ssh
	"SSH_ASKPASS",         // custom ssh askpass helper
	"XDG_RUNTIME_DIR",     // required by ssh-agent
	"USER",                // required by git
	"HOME",                // required by git
	"GIT_TERMINAL_PROMPT", // to disable git terminal prompts
	"GIT_ASKPASS",         // custom git askpass helper
	"GIT_SSH_COMMAND",     // to pass custom ssh options
	"GIT_SSH",             // to pass custom ssh options (legacy)
	"GH_TOKEN",            // GitHub token for GitHub CLI as credential helper
	"GITHUB_TOKEN",        // GitHub token for GitHub CLI as credential helper
	"TMPDIR",              // required by git for temporary files
	"TEMP",                // required by git for temporary files on Windows
	"TMP",                 // required by git for temporary files on Windows
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
		if val, ok := os.LookupEnv(key); ok { //nolint:forbidigo
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

	// If k6repo is a versioned k6 module path (e.g. go.k6.io/k6/v2), extract the major
	// version so k6foundry can resolve the correct module path for non-semver versions
	// such as "latest". For actual forks (e.g. github.com/myfork/k6), set K6Repo instead.
	if strings.HasPrefix(opts.k6repo, defaultK6Repo+"/v") {
		fopts.K6MajorVersion = opts.k6repo[len(defaultK6Repo+"/"):]
	} else if opts.k6repo != defaultK6Repo {
		fopts.K6Repo = opts.k6repo
	}

	if logger.Enabled(ctx, slog.LevelDebug) {
		fopts.Stdout = os.Stdout //nolint:forbidigo
		fopts.Stderr = os.Stderr //nolint:forbidigo
	}

	return k6foundry.NewNativeFoundry(ctx, fopts)
}

// findK6ModVersion looks up the k6 module (any major version) in the given ModVersions map.
// It returns the module path, version, and whether it was found.
func findK6ModVersion(modVersions map[string]string) (path, version string, found bool) {
	for k, v := range modVersions {
		if k == defaultK6Repo || strings.HasPrefix(k, defaultK6Repo+"/v") {
			return k, v, true
		}
	}

	return "", "", false
}

func buildK6(ctx context.Context, opts *buildOptions) (*k6foundry.BuildInfo, error) {
	// When using the default k6 repo, resolve the correct module path so that
	// v2+ releases are handled without requiring --k6-repo.
	resolveK6Repo(ctx, opts)

	foundry, err := newFoundry(ctx, opts)
	if err != nil {
		return nil, err
	}

	const outFilePerm = 0o777

	out, err := os.OpenFile(opts.output, os.O_WRONLY|os.O_CREATE, outFilePerm) /* #nosec G302 G304 */ //nolint:forbidigo
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

	err = out.Close()
	if err != nil {
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

// resolveK6Repo sets opts.k6repo (and opts.k6version when appropriate) to the
// correct k6 module path when the user has not overridden --k6-repo. This
// ensures that v2+ releases are used without requiring an explicit flag.
func resolveK6Repo(ctx context.Context, opts *buildOptions) {
	if opts.k6repo != defaultK6Repo {
		slog.Debug("Using explicit k6 repo, skipping auto-resolution", "repo", opts.k6repo)
		return
	}

	if opts.k6version == defaultK6Version {
		slog.Debug("Resolving k6 module from extension dependencies (version: latest)")

		// No explicit version: inspect extension dependencies first, then
		// fall back to the overall latest across all major versions.
		path, version, err := sync.ResolveK6ModuleForExtensions(ctx, extensionModules(opts))
		if err != nil {
			slog.Warn("Failed to resolve k6 module from extensions, using default", "error", err)
			return
		}

		slog.Debug("Resolved k6 module", "repo", path, "version", version)

		opts.k6repo = path
		opts.k6version = version

		return
	}

	slog.Debug("Resolving k6 module path for explicit version", "version", opts.k6version)

	// Explicit version (semver, SHA, branch name): detect the module path from
	// the version itself so that e.g. a v2 SHA resolves to go.k6.io/k6/v2.
	path, err := sync.ResolveK6ModuleForVersion(ctx, opts.k6version)
	if err != nil {
		slog.Warn("Failed to resolve k6 module for version, using default", "error", err)
		return
	}

	slog.Debug("Resolved k6 module path", "repo", path, "version", opts.k6version)

	opts.k6repo = path
}

func extensionModules(opts *buildOptions) []sync.ExtensionModule {
	exts := make([]sync.ExtensionModule, 0, len(opts.extensions.modules))
	for _, m := range opts.extensions.modules {
		exts = append(exts, sync.ExtensionModule{
			Path:      m.Path,
			Version:   m.Version,
			LocalPath: m.ReplacePath,
		})
	}

	return exts
}

func (m *modules) Type() string {
	if m.replace {
		return "module=replacement"
	}

	return "module[@version][=replacement]"
}
