package k6foundry

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultWorkDir = "k6foundry*"

	mainModuleTemplate = `package main

import (
	k6cmd "%s/cmd"

)

func main() {
	k6cmd.Execute()
}
`
	modImportTemplate = `package main

	import _ %q
`
)

// native is a foundry backed by th golang toolchain
type native struct {
	NativeFoundryOpts
	log *slog.Logger
}

// NativeFoundryOpts defines the options for the Native foundry
type NativeFoundryOpts struct {
	// options used for running go
	GoOpts
	// use alternative k6 repository
	K6Repo string
	// K6MajorVersion overrides the k6 major version used to determine the module path.
	// Only used when the k6Version passed to Build is not a valid semver (e.g. "latest" or a commit SHA).
	// Example: set to "v2" to build against go.k6.io/k6/v2@latest.
	K6MajorVersion string
	// don't cleanup work environment (useful for debugging)
	SkipCleanup bool
	// redirect stdout
	Stdout io.Writer
	// redirect stderr
	Stderr io.Writer
	// set log level (INFO, WARN, ERROR)
	Logger *slog.Logger
}

// NewDefaultNativeFoundry creates a new native build environment with default options
func NewDefaultNativeFoundry() (Foundry, error) {
	return NewNativeFoundry(
		context.TODO(),
		NativeFoundryOpts{
			GoOpts: GoOpts{
				CopyGoEnv: true,
			},
		},
	)
}

// NewNativeFoundry creates a new native build environment with the given options
func NewNativeFoundry(_ context.Context, opts NativeFoundryOpts) (Foundry, error) {
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}

	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}

	// set default logger if none passed
	log := opts.Logger
	if log == nil {
		log = slog.New(
			slog.NewTextHandler(
				opts.Stderr,
				&slog.HandlerOptions{},
			),
		)
	}

	return &native{
		NativeFoundryOpts: opts,
		log:               log,
	}, nil
}

// Build builds a custom k6 binary for a target platform with the given dependencies into the out io.Writer
func (b *native) Build(
	ctx context.Context,
	platform Platform,
	k6Version string,
	exts []Module,
	replacements []Module,
	buildOpts []string,
	binary io.Writer,
) (*BuildInfo, error) {
	workDir, err := os.MkdirTemp(os.TempDir(), defaultWorkDir) //nolint:forbidigo
	if err != nil {
		return nil, fmt.Errorf("creating working directory: %w", err)
	}

	defer b.cleanupWorkDir(workDir)

	// prepare the build environment
	b.log.Info("Building new k6 binary (native)")

	k6Binary := filepath.Join(workDir, "k6")

	buildEnv, err := newGoEnv(workDir,
		b.GoOpts,
		platform,
		b.Stdout,
		b.Stderr,
	)
	if err != nil {
		return nil, err
	}

	defer b.cleanupBuildEnv(ctx, buildEnv)

	buildInfo := newBuildInfo(platform.String())
	b.log.Info("Initializing Go module")
	err = buildEnv.modInit(ctx)
	if err != nil {
		return nil, err
	}

	// apply replacements
	for _, r := range replacements {
		err = b.addReplacement(ctx, buildEnv, r)
		if err != nil {
			return nil, err
		}
	}

	k6ModPath, err := k6ModulePath(k6Version, b.K6MajorVersion)
	if err != nil {
		return nil, fmt.Errorf("determining k6 module path: %w", err)
	}

	b.log.Info("Creating k6 main")
	err = b.createMain(ctx, workDir, k6ModPath)
	if err != nil {
		return nil, err
	}

	err = b.handleK6Overrides(ctx, k6Version, k6ModPath, buildEnv, buildInfo)
	if err != nil {
		return nil, err
	}

	b.log.Info("importing extensions")
	err = b.importModules(ctx, exts, workDir, buildEnv, buildInfo)
	if err != nil {
		return nil, err
	}

	b.warnK6VersionConflicts(ctx, k6ModPath, buildEnv, buildInfo)

	b.log.Info("Building k6")
	err = buildEnv.compile(ctx, k6Binary, buildOpts...)
	if err != nil {
		return nil, err
	}

	b.log.Info("Build complete")
	if err = b.copyBinary(k6Binary, binary); err != nil {
		return nil, err
	}

	return buildInfo, nil
}

func (b *native) copyBinary(path string, dst io.Writer) error {
	f, err := os.Open(path) //nolint:gosec,forbidigo
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, f)
	if err != nil {
		return fmt.Errorf("copying binary %w", err)
	}
	return nil
}

func (b *native) cleanupWorkDir(workDir string) {
	if b.SkipCleanup {
		b.log.Info(fmt.Sprintf("Skipping cleanup. leaving directory %s intact", workDir))
		return
	}

	b.log.Info(fmt.Sprintf("Cleaning up work directory %s", workDir))
	_ = os.RemoveAll(workDir) //nolint:forbidigo
}

func (b *native) cleanupBuildEnv(ctx context.Context, buildEnv *goEnv) {
	if b.SkipCleanup {
		b.log.Info("Skipping go cleanup")
		return
	}
	_ = buildEnv.close(ctx)
}

func (b *native) handleK6Overrides(
	ctx context.Context, k6Version, k6ModPath string, buildEnv *goEnv, buildInfo *BuildInfo,
) error {
	k6ReplaceVersion := ""
	if b.K6Repo != "" {
		k6ReplaceVersion = k6Version
		k6Version = ""
	}
	k6Mod := Module{
		Path:           k6ModPath,
		Version:        k6Version,
		ReplacePath:    b.K6Repo,
		ReplaceVersion: k6ReplaceVersion,
	}

	modVer, err := b.addMod(ctx, buildEnv, k6Mod)
	if err != nil {
		return err
	}

	buildInfo.ModVersions[k6ModPath] = modVer
	return nil
}

func (b *native) importModules(
	ctx context.Context, exts []Module, workDir string, buildEnv *goEnv, buildInfo *BuildInfo,
) error {
	for _, m := range exts {
		err := b.createModuleImport(ctx, workDir, m)
		if err != nil {
			return err
		}

		modVer, err := b.addMod(ctx, buildEnv, m)
		if err != nil {
			return err
		}
		buildInfo.ModVersions[m.Path] = modVer
	}
	return nil
}

func (b *native) createMain(_ context.Context, path string, k6ModPath string) error {
	// write the main module file
	mainPath := filepath.Join(path, "main.go")
	mainContent := fmt.Sprintf(mainModuleTemplate, k6ModPath)
	err := os.WriteFile(mainPath, []byte(mainContent), 0o600) //nolint:forbidigo
	if err != nil {
		return fmt.Errorf("writing main file %w", err)
	}

	return nil
}

func (b *native) addReplacement(ctx context.Context, e *goEnv, rep Module) error {
	if rep.ReplacePath == "" {
		return fmt.Errorf("replace path is required")
	}

	b.log.Info(fmt.Sprintf("replacing dependency %s", rep.String()))

	// resolve path to and absolute path because the mod replace will occur in the work directory
	replacePath, err := resolvePath(rep.ReplacePath)
	if err != nil {
		return fmt.Errorf("resolving replace path: %w", err)
	}

	return e.modReplace(ctx, rep.Path, rep.Version, replacePath, rep.ReplaceVersion)
}

func (b *native) addMod(ctx context.Context, e *goEnv, mod Module) (string, error) {
	b.log.Info(fmt.Sprintf("adding dependency %s", mod.String()))

	if mod.ReplacePath == "" {
		if err := e.modRequire(ctx, mod.Path, mod.Version); err != nil {
			return "", err
		}

		if err := e.modTidy(ctx); err != nil {
			return "", err
		}

		return e.modVersion(ctx, mod.Path)
	}

	// resolve path to and absolute path because the mod replace will occur in the work directory
	replacePath, err := resolvePath(mod.ReplacePath)
	if err != nil {
		return "", fmt.Errorf("resolving replace path: %w", err)
	}

	if err := e.modReplace(ctx, mod.Path, mod.Version, replacePath, mod.ReplaceVersion); err != nil {
		return "", err
	}

	if err := e.modTidy(ctx); err != nil {
		return "", err
	}

	return e.modVersion(ctx, mod.Path)
}

func resolvePath(path string) (string, error) {
	var err error
	// expand environment variables
	if strings.Contains(path, "$") {
		path = os.ExpandEnv(path) //nolint:forbidigo
	}

	if strings.HasPrefix(path, ".") {
		path, err = filepath.Abs(path)
		if err != nil {
			return "", err
		}
	}

	return path, nil
}

// warnK6VersionConflicts checks the resolved module graph for k6 major versions other than
// the one being built. This occurs when an extension depends on a different k6 major version
// (e.g. a v1 extension used in a k6 v2 build). The build succeeds but those extensions will
// silently register with an inactive k6 runtime and have no effect at runtime.
func (b *native) warnK6VersionConflicts(ctx context.Context, k6ModPath string, buildEnv *goEnv, buildInfo *BuildInfo) {
	modules, err := buildEnv.listModules(ctx)
	if err != nil {
		b.log.Warn(fmt.Sprintf("could not check for k6 version conflicts: %s", err))
		return
	}

	for _, mod := range modules {
		if mod == k6ModPath {
			continue
		}
		if mod == k6BaseModulePath || strings.HasPrefix(mod, k6BaseModulePath+"/v") {
			msg := fmt.Sprintf(
				"conflicting k6 versions detected: building %s but %s is also in the module graph; "+
					"extensions depending on %s will not be active",
				k6ModPath, mod, mod,
			)
			b.log.Warn(msg)
			buildInfo.Warnings = append(buildInfo.Warnings, Warning{
				Code:    WarnK6VersionConflict,
				Message: msg,
			})
		}
	}
}

func (b *native) createModuleImport(_ context.Context, path string, mod Module) error {
	modImportFile := filepath.Join(path, strings.ReplaceAll(mod.Path, "/", "_")+".go")
	modImportContent := fmt.Sprintf(modImportTemplate, mod.Path)
	err := os.WriteFile(modImportFile, []byte(modImportContent), 0o600) //nolint:forbidigo
	if err != nil {
		return fmt.Errorf("writing mod file %w", err)
	}

	return nil
}
