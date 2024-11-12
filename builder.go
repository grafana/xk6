package xk6

import (
	"context"
	"fmt"
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

// Builder can produce a custom k6 build with the
// configuration it represents.
type Builder struct {
	Compile
	K6Repo       string        `json:"k6_repo,omitempty"`
	K6Version    string        `json:"k6_version,omitempty"`
	Extensions   []Dependency  `json:"extensions,omitempty"`
	Replacements []Replace     `json:"replacements,omitempty"`
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
// XK6_K6_REPO sets the path to the main k6 repository. This is useful when building with k6 forks
func FromOSEnv() Builder {
	env := map[string]string{}
	for _, arg := range os.Environ() {
		parts := strings.SplitN(arg, "=", 2)
		env[parts[0]] = parts[1]
	}
	return parseEnv(env)
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

// Build builds k6 at the configured version with the
// configured extensions and writes a binary at outputFile.
func (b Builder) Build(ctx context.Context, log *slog.Logger, outfile string) error {
	if outfile == "" {
		return fmt.Errorf("output file path is required")
	}

	// set some defaults from the environment, if applicable
	if b.OS == "" {
		b.OS = runtime.GOOS
	}
	if b.Arch == "" {
		b.Arch = runtime.GOARCH
	}

	// We are not passing all the current environment anymore! ONLY the GO environment
	// env := os.Environ()
	env := map[string]string{
		"GOARM": b.ARM,
	}

	raceArg := "-race"

	// trim debug symbols by default
	if (b.RaceDetector || strings.Contains(b.BuildFlags, raceArg)) && !b.Compile.Cgo {
		log.Warn("Enabling cgo because it is required by the race detector")
		b.Compile.Cgo = true
	}
	env["CGO_ENABLED"] = b.Compile.CgoEnabled()

	log.Info("Building k6")

	opts := k6foundry.NativeBuilderOpts{
		GoOpts: k6foundry.GoOpts{
			GoGetTimeout:   b.TimeoutGet,
			GOBuildTimeout: b.TimeoutBuild,
			CopyGoEnv:      true,
			Env:            env,
		},
		SkipCleanup: b.SkipCleanup,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Logger:      log,
	}

	k6b, err := k6foundry.NewNativeBuilder(ctx, opts)
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
	outFile, err := os.OpenFile(absOutputFile, os.O_WRONLY|os.O_CREATE, 0o777) //nolint:gosec
	if err != nil {
		return err
	}
	defer outFile.Close() //nolint:errcheck

	platform, err := k6foundry.ParsePlatform(b.OS + "/" + b.Arch)
	if err != nil {
		return err
	}

	mods := []k6foundry.Module{}
	for _, e := range b.Extensions {
		mod := k6foundry.Module{Path: e.PackagePath, Version: e.Version}
		if mod.Version == "" {
			mod.Version = "latest"
		}
		mods = append(mods, mod)
	}
	for _, r := range b.Replacements {
		// parse again the replacement, ignore the error because it was already parsed
		mod, _ := k6foundry.ParseModule(fmt.Sprintf("%s=%s", r.Old.String(), r.New.String()))
		mods = append(mods, mod)
	}

	k6Version := b.K6Version
	if k6Version == "" {
		k6Version = "latest"
	}
	_, err = k6b.Build(ctx, platform, k6Version, mods, buildCommandArgs(b.BuildFlags), outFile)
	return err
}

// Dependency pairs a Go module path with a version.
type Dependency struct {
	// The name (import path) of the Go package. If at a version > 1,
	// it should contain semantic import version (i.e. "/v2").
	// Used with `go get`.
	PackagePath string `json:"package_path,omitempty"`

	// The version of the Go module, as used with `go get`.
	Version string `json:"version,omitempty"`
}

// ReplacementPath represents an old or new path component
// within a Go module replacement directive.
type ReplacementPath string

// Param reformats a go.mod replace directive to be
// compatible with the `go mod edit` command.
func (r ReplacementPath) Param() string {
	return strings.Replace(string(r), " ", "@", 1)
}

func (r ReplacementPath) String() string { return string(r) }

// Replace represents a Go module replacement.
type Replace struct {
	// The import path of the module being replaced.
	Old ReplacementPath `json:"old,omitempty"`

	// The path to the replacement module.
	New ReplacementPath `json:"new,omitempty"`
}

// NewReplace creates a new instance of Replace provided old and
// new Go module paths
func NewReplace(old, new string) Replace {
	return Replace{
		Old: ReplacementPath(old),
		New: ReplacementPath(new),
	}
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
// so we may pass args separately to newCommand()
func buildCommandArgs(buildFlags string) []string {
	buildFlagsSlice := make([]string, 0, 10)

	tmp := []string{}
	sb := &strings.Builder{}
	quoted := false
	for _, r := range buildFlags {
		if r == '"' || r == '\'' {
			quoted = !quoted
		} else if !quoted && r == ' ' {
			tmp = append(tmp, sb.String())
			sb.Reset()
		} else {
			sb.WriteRune(r)
		}
	}
	if sb.Len() > 0 {
		tmp = append(tmp, sb.String())
	}

	buildFlagsSlice = append(buildFlagsSlice, tmp...)

	return buildFlagsSlice
}
