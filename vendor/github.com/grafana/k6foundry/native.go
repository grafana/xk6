//nolint:forbidigo,revive,funlen
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
	defaultK6ModulePath = "go.k6.io/k6"

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
	workDir, err := os.MkdirTemp(os.TempDir(), defaultWorkDir)
	if err != nil {
		return nil, fmt.Errorf("creating working directory: %w", err)
	}

	defer func() {
		if b.SkipCleanup {
			b.log.Info(fmt.Sprintf("Skipping cleanup. leaving directory %s intact", workDir))
			return
		}

		b.log.Info(fmt.Sprintf("Cleaning up work directory %s", workDir))
		_ = os.RemoveAll(workDir)
	}()

	// prepare the build environment
	b.log.Info("Building new k6 binary (native)")

	k6Binary := filepath.Join(workDir, "k6")

	buildEnv, err := newGoEnv(
		workDir,
		b.GoOpts,
		platform,
		b.Stdout,
		b.Stderr,
	)
	if err != nil {
		return nil, err
	}

	defer func() {
		if b.SkipCleanup {
			b.log.Info("Skipping go cleanup")
			return
		}
		_ = buildEnv.close(ctx)
	}()

	buildInfo := &BuildInfo{
		Platform:    platform.String(),
		ModVersions: map[string]string{},
	}

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

	b.log.Info("Creating k6 main")
	err = b.createMain(ctx, workDir)
	if err != nil {
		return nil, err
	}

	k6ReplaceVersion := ""
	if b.K6Repo != "" {
		k6ReplaceVersion = k6Version
		k6Version = ""
	}
	k6Mod := Module{
		Path:           defaultK6ModulePath,
		Version:        k6Version,
		ReplacePath:    b.K6Repo,
		ReplaceVersion: k6ReplaceVersion,
	}

	modVer, err := b.addMod(ctx, buildEnv, k6Mod)
	if err != nil {
		return nil, err
	}

	buildInfo.ModVersions[defaultK6ModulePath] = modVer

	b.log.Info("importing extensions")
	for _, m := range exts {
		err = b.createModuleImport(ctx, workDir, m)
		if err != nil {
			return nil, err
		}

		modVer, err = b.addMod(ctx, buildEnv, m)
		if err != nil {
			return nil, err
		}
		buildInfo.ModVersions[m.Path] = modVer
	}

	b.log.Info("Building k6")
	err = buildEnv.compile(ctx, k6Binary, buildOpts...)
	if err != nil {
		return nil, err
	}

	b.log.Info("Build complete")
	k6File, err := os.Open(k6Binary) //nolint:gosec
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(binary, k6File)
	if err != nil {
		return nil, fmt.Errorf("copying binary %w", err)
	}

	return buildInfo, nil
}

func (b *native) createMain(_ context.Context, path string) error {
	// write the main module file
	mainPath := filepath.Join(path, "main.go")
	mainContent := fmt.Sprintf(mainModuleTemplate, defaultK6ModulePath)
	err := os.WriteFile(mainPath, []byte(mainContent), 0o600)
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
		path = os.ExpandEnv(path)
	}

	if strings.HasPrefix(path, ".") {
		path, err = filepath.Abs(path)
		if err != nil {
			return "", err
		}
	}

	return path, nil
}

func (b *native) createModuleImport(_ context.Context, path string, mod Module) error {
	modImportFile := filepath.Join(path, strings.ReplaceAll(mod.Path, "/", "_")+".go")
	modImportContent := fmt.Sprintf(modImportTemplate, mod.Path)
	err := os.WriteFile(modImportFile, []byte(modImportContent), 0o600)
	if err != nil {
		return fmt.Errorf("writing mod file %w", err)
	}

	return nil
}
