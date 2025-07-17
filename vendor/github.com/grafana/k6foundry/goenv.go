//nolint:revive,forbidigo
package k6foundry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var (
	// Error compiling binary
	ErrCompiling = errors.New("compiling")
	// Error executing go command
	ErrExecutingGoCommand = errors.New("executing go command")
	// Go toolchacin is not installed
	ErrNoGoToolchain = errors.New("go toolchain notfound")
	// Git is not installed
	ErrNoGit = errors.New("git notfound")
	// Error resolving dependency
	ErrResolvingDependency = errors.New("resolving dependency")
	// Error initiailizing go build environment
	ErrSettingGoEnv = errors.New("setting go environment")

	goVersionRegex = regexp.MustCompile(`go1(\.[0-9]+){1,3}(-[a-f0-9]+)?`)
)

// GoOpts defines the options for the go build environment
type GoOpts struct {
	// Environment variables passed to the build service
	// Can override variables copied from the current go environment
	Env map[string]string
	// Copy Environment variables to go build environment
	CopyGoEnv bool
	// Timeout for getting modules
	GoGetTimeout time.Duration
	// Timeout for building binary
	GOBuildTimeout time.Duration
	// Use an ephemeral cache. Ignores GoModCache and GoCache
	TmpCache bool
}

type goEnv struct {
	env          []string
	workDir      string
	platform     Platform
	stdout       io.Writer
	stderr       io.Writer
	tmpDirs      []string
	tmpCache     bool
	buildTimeout time.Duration
	getTimeout   time.Duration
}

func newGoEnv(
	workDir string,
	opts GoOpts,
	platform Platform,
	stdout io.Writer,
	stderr io.Writer,
) (*goEnv, error) {
	var (
		err     error
		tmpDirs []string
	)

	if _, hasGo := goVersion(); !hasGo {
		return nil, ErrNoGoToolchain
	}

	if !hasGit() {
		return nil, ErrNoGit
	}

	env := map[string]string{}

	// copy current go environment
	if opts.CopyGoEnv {
		env, err = getGoEnv()
		if err != nil {
			return nil, fmt.Errorf("copying go environment %w", err)
		}
	}

	// set/override environment variables
	maps.Copy(env, opts.Env)

	if opts.TmpCache {
		// override caches with temporary files
		var modCache, goCache string
		modCache, err = os.MkdirTemp(os.TempDir(), "modcache*")
		if err != nil {
			return nil, fmt.Errorf("creating mod cache %w", err)
		}

		goCache, err = os.MkdirTemp(os.TempDir(), "cache*")
		if err != nil {
			return nil, fmt.Errorf("creating go cache %w", err)
		}

		env["GOCACHE"] = goCache
		env["GOMODCACHE"] = modCache

		// add to the list of directories for cleanup
		tmpDirs = append(tmpDirs, goCache, modCache)
	}

	// ensure path is set
	env["PATH"] = os.Getenv("PATH")

	// override platform
	env["GOOS"] = platform.OS
	env["GOARCH"] = platform.Arch

	// disable CGO if target platform is different from host platform
	if env["GOHOSTARCH"] != platform.Arch || env["GOHOSTOS"] != platform.OS {
		env["CGO_ENABLED"] = "0"
	}

	return &goEnv{
		env:          mapToSlice(env),
		platform:     platform,
		workDir:      workDir,
		stdout:       stdout,
		stderr:       stderr,
		buildTimeout: opts.GOBuildTimeout,
		getTimeout:   opts.GoGetTimeout,
		tmpDirs:      tmpDirs,
		tmpCache:     opts.TmpCache,
	}, nil
}

func (e goEnv) close(ctx context.Context) error {
	var err error

	if e.tmpCache {
		// clean caches, otherwirse directories can't be deleted
		err = e.clean(ctx)
	}

	// creal all temporary dirs
	for _, dir := range e.tmpDirs {
		err = errors.Join(
			err,
			os.RemoveAll(dir),
		)
	}

	return err
}

func (e goEnv) runGo(ctx context.Context, timeout time.Duration, args ...string) error {
	cmd := exec.Command("go", args...)

	cmd.Env = e.env
	cmd.Dir = e.workDir

	cmd.Stdout = e.stdout
	cmd.Stderr = e.stderr

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// start the command; if it fails to start, report error immediately
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("%w: %s", ErrExecutingGoCommand, err.Error())
	}

	// wait for the command in a goroutine; the reason for this is
	// very subtle: if, in our select, we do `case cmdErr := <-cmd.Wait()`,
	// then that case would be chosen immediately, because cmd.Wait() is
	// immediately available (even though it blocks for potentially a long
	// time, it can be evaluated immediately). So we have to remove that
	// evaluation from the `case` statement.
	cmdErrChan := make(chan error)
	go func() {
		cmdErr := cmd.Wait()
		if cmdErr != nil {
			cmdErr = fmt.Errorf("%w: %s", ErrExecutingGoCommand, cmdErr.Error())
		}
		cmdErrChan <- cmdErr
	}()

	// unblock either when the command finishes, or when the done
	// channel is closed -- whichever comes first
	select {
	case cmdErr := <-cmdErrChan:
		// process ended; report any error immediately
		return cmdErr
	case <-ctx.Done():
		// context was canceled, either due to timeout or
		// maybe a signal from higher up canceled the parent
		// context; presumably, the OS also sent the signal
		// to the child process, so wait for it to die
		select {
		// TODO: check this magic timeout
		case <-time.After(15 * time.Second):
			_ = cmd.Process.Kill()
		case <-cmdErrChan:
		}
		return ctx.Err()
	}
}

func (e goEnv) modInit(ctx context.Context) error {
	// initialize the go module
	// TODO: change magic constant in timeout
	err := e.runGo(ctx, 10*time.Second, "mod", "init", "k6")
	if err != nil {
		return fmt.Errorf("%w: %s", ErrSettingGoEnv, err.Error())
	}

	return nil
}

// tidy the module to ensure go.mod will not have versions such as `latest`
func (e goEnv) modTidy(ctx context.Context) error {
	err := e.runGo(ctx, e.getTimeout, "mod", "tidy", "-compat=1.17")
	if err != nil {
		return fmt.Errorf("%w: %s", ErrResolvingDependency, err.Error())
	}

	return nil
}

func (e goEnv) modRequire(ctx context.Context, modulePath, moduleVersion string) error {
	if moduleVersion == "" {
		moduleVersion = "latest"
	}

	modulePath += "@" + moduleVersion

	err := e.runGo(ctx, e.getTimeout, "mod", "edit", "-require", modulePath)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrResolvingDependency, err.Error())
	}

	return nil
}

func (e goEnv) modReplace(ctx context.Context, modulePath, moduleVersion, replacePath, replaceVersion string) error {
	if moduleVersion != "" && moduleVersion != "latest" {
		modulePath += "@" + moduleVersion
	}

	if replaceVersion != "" {
		replacePath += "@" + replaceVersion
	}

	err := e.runGo(ctx, e.getTimeout, "mod", "edit", "-replace", fmt.Sprintf("%s=%s", modulePath, replacePath))
	if err != nil {
		return fmt.Errorf("%w: %s", ErrResolvingDependency, err.Error())
	}

	return e.modTidy(ctx)
}

func (e goEnv) compile(ctx context.Context, outPath string, buildFlags ...string) error {
	args := append([]string{"build", "-o", outPath}, buildFlags...)

	err := e.runGo(ctx, e.buildTimeout, args...)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrCompiling, err.Error())
	}

	return err
}

func (e goEnv) clean(ctx context.Context) error {
	err := e.runGo(ctx, e.buildTimeout, "clean", "-cache", "-modcache")
	if err != nil {
		return fmt.Errorf("cleaning: %s", err.Error())
	}

	return err
}

func (e goEnv) modVersion(_ context.Context, mod string) (string, error) {
	// can't use runGo because we need the output
	cmd := exec.Command("go", "list", "-f", "{{.Version}} {{.Replace}}", "-m", mod)
	cmd.Env = e.env
	cmd.Dir = e.workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("list module %s", err.Error())
	}

	// go list will return the version and the replacement if any
	result := strings.Split(strings.Trim(string(out), "\n"), " ")
	version := result[0]

	// if there's a replacement, use the replacement's version
	if len(result) == 3 {
		version = result[2]
	}

	return version, nil
}

func mapToSlice(m map[string]string) []string {
	s := []string{}
	for k, v := range m {
		s = append(s, fmt.Sprintf("%s=%s", k, v))
	}

	return s
}

func goVersion() (string, bool) {
	cmd, err := exec.LookPath("go")
	if err != nil {
		return "", false
	}

	out, err := exec.Command(cmd, "version").Output() //nolint:gosec
	if err != nil {
		return "", false
	}

	ver := goVersionRegex.Find(out)
	if ver == nil {
		return "", false
	}

	return string(ver), true
}

func getGoEnv() (map[string]string, error) {
	cmd, err := exec.LookPath("go")
	if err != nil {
		return nil, fmt.Errorf("getting go binary %w", err)
	}

	out, err := exec.Command(cmd, "env", "-json").Output() //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("getting go env %w", err)
	}

	envMap := map[string]string{}

	err = json.Unmarshal(out, &envMap)
	if err != nil {
		return nil, fmt.Errorf("getting go env %w", err)
	}

	return envMap, err
}

func hasGit() bool {
	cmd, err := exec.LookPath("git")
	if err != nil {
		return false
	}

	_, err = exec.Command(cmd, "version").Output() //nolint:gosec

	return err == nil
}
