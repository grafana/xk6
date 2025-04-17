// Package legacy contains the legacy xk6 builder API.
package legacy

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/grafana/k6foundry"
)

const (
	defaultBuildFlags = "-ldflags='-w -s' -trimpath"
)

var errMissingOutputFile = errors.New("output file path is required")

// Builder can produce a custom k6 build with the
// configuration it represents.
type Builder struct {
	Compile
	K6Repo       string        `json:"k6_repo,omitempty"`
	K6Version    string        `json:"k6_version,omitempty"`
	Extensions   []string      `json:"extensions,omitempty"`
	Replacements []string      `json:"replacements,omitempty"`
	TimeoutGet   time.Duration `json:"timeout_get,omitempty"`
	TimeoutBuild time.Duration `json:"timeout_build,omitempty"`
	RaceDetector bool          `json:"race_detector,omitempty"`
	SkipCleanup  bool          `json:"skip_cleanup,omitempty"`
	BuildFlags   string        `json:"build_flags,omitempty"`
}

// FromOSEnv creates a Builder from environment variables:
// GOARCH, GOOS, GOARM defines builder's target platform
// K6_VERSION sets the version of k6 to build.
// XK6_BUILD_FLAGS sets any go build flags if needed. Defaults to '-ldflags=-w -s -trim'.
// XK6_RACE_DETECTOR enables the Go race detector in the build. Forces CGO_ENABLED=1
// XK6_SKIP_CLEANUP causes xk6 to leave build artifacts on disk after exiting.
// XK6_K6_REPO sets the path to the main k6 repository. This is useful when building with k6 forks.
func FromOSEnv() Builder {
	env := map[string]string{}

	const assignParts = 2

	for _, arg := range os.Environ() {
		parts := strings.SplitN(arg, "=", assignParts)
		env[parts[0]] = parts[1]
	}

	return parseEnv(env)
}

// Build builds k6 at the configured version with the
// configured extensions and writes a binary at outputFile.
func (b Builder) Build(ctx context.Context, log *slog.Logger, outfile string) error {
	if outfile == "" {
		return errMissingOutputFile
	}

	log.Info("Building k6")

	opts := k6foundry.NativeFoundryOpts{
		GoOpts: k6foundry.GoOpts{
			GoGetTimeout:   b.TimeoutGet,
			GOBuildTimeout: b.TimeoutBuild,
			CopyGoEnv:      true,
			Env:            b.newBuilderEnv(log),
		},
		K6Repo:      b.K6Repo,
		SkipCleanup: b.SkipCleanup,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Logger:      log,
	}

	k6b, err := k6foundry.NewNativeFoundry(ctx, opts)
	if err != nil {
		return err
	}

	// the user's specified output file might be relative, and
	// because the `go build` command is executed in a different,
	// temporary folder, we convert the user's input to an
	// absolute path so it goes the expected place
	absOutputFile, err := filepath.Abs(outfile)
	if err != nil {
		return err
	}

	const outFilePerm = 0o777

	outFile, err := os.OpenFile(absOutputFile, os.O_WRONLY|os.O_CREATE, outFilePerm) // #nosec G302 G304
	if err != nil {
		return err
	}
	defer outFile.Close() //nolint:errcheck

	platform, err := b.parsePlatform()
	if err != nil {
		return err
	}

	mods, err := b.getModules()
	if err != nil {
		return err
	}

	reps, err := b.getReplacements()
	if err != nil {
		return err
	}

	k6Version := b.K6Version
	if k6Version == "" {
		k6Version = "latest"
	}

	_, err = k6b.Build(ctx, platform, k6Version, mods, reps, buildCommandArgs(b.BuildFlags), outFile)

	return err
}

func parseEnv(env map[string]string) Builder {
	return Builder{
		Compile: Compile{
			Platform: Platform{
				OS:   env["GOOS"],
				Arch: env["GOARCH"],
				ARM:  env["GOARM"],
			},
		},
		K6Version:    env["K6_VERSION"],
		K6Repo:       env["XK6_K6_REPO"],
		RaceDetector: env["XK6_RACE_DETECTOR"] == "1",
		SkipCleanup:  env["XK6_SKIP_CLEANUP"] == "1",
		BuildFlags:   envOrDefaultValue(env, "XK6_BUILD_FLAGS", defaultBuildFlags),
	}
}

func osEnv() map[string]string {
	env := make(map[string]string)

	for _, entry := range os.Environ() {
		key, val, _ := strings.Cut(entry, "=")
		env[key] = val
	}

	return env
}

func (b Builder) newBuilderEnv(log *slog.Logger) map[string]string {
	env := osEnv()

	env["GOARM"] = b.ARM

	raceArg := "-race"

	// trim debug symbols by default
	if (b.RaceDetector || strings.Contains(b.BuildFlags, raceArg)) && !b.Cgo {
		log.Warn("Enabling cgo because it is required by the race detector")

		b.Cgo = true
	}

	env["CGO_ENABLED"] = b.CgoEnabled()

	return env
}

func (b Builder) getModules() ([]k6foundry.Module, error) {
	mods := []k6foundry.Module{}

	for _, e := range b.Extensions {
		mod, err := k6foundry.ParseModule(e)
		if err != nil {
			return []k6foundry.Module{}, err
		}

		mods = append(mods, mod)
	}

	return mods, nil
}

func (b Builder) getReplacements() ([]k6foundry.Module, error) {
	reps := []k6foundry.Module{}

	for _, r := range b.Replacements {
		rep, err := k6foundry.ParseModule(r)
		if err != nil {
			return []k6foundry.Module{}, err
		}

		reps = append(reps, rep)
	}

	return reps, nil
}

func (b Builder) parsePlatform() (k6foundry.Platform, error) {
	os := b.OS
	if os == "" {
		os = runtime.GOOS
	}

	arch := b.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}

	return k6foundry.ParsePlatform(os + "/" + arch)
}

func envOrDefaultValue(env map[string]string, name, defaultValue string) string {
	s, ok := env[name]
	if !ok {
		return defaultValue
	}

	return s
}

// buildCommandArgs parses the build flags passed by environment variable XK6_BUILD_FLAGS
// or the default values when no value for it is given
// so we may pass args separately to newCommand().
func buildCommandArgs(buildFlags string) []string {
	const avgNumberOfFlags = 10
	buildFlagsSlice := make([]string, 0, avgNumberOfFlags)

	tmp := []string{}
	sb := &strings.Builder{}
	quoted := false

	for _, r := range buildFlags {
		switch {
		case r == '"' || r == '\'':
			quoted = !quoted
		case !quoted && r == ' ':
			tmp = append(tmp, sb.String())
			sb.Reset()
		default:
			sb.WriteRune(r)
		}
	}

	if sb.Len() > 0 {
		tmp = append(tmp, sb.String())
	}

	buildFlagsSlice = append(buildFlagsSlice, tmp...)

	return buildFlagsSlice
}
