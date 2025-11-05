package lint

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"golang.org/x/mod/modfile"
)

type stateKey struct{}

// state caches computations during the linting process to avoid redundant work.
// Design principles:
//
//   - checkers may use state but it's not required
//   - checkers are read-only and never modify state
//   - getter methods return cached values or compute, cache, and return new values
type state struct {
	dir string

	_moduleFileCached    *modfile.File
	_exePathCached       string
	_versionOutputCached []byte
	_isJSCached          *bool
	_hasExtensionCached  *bool
}

//nolint:gochecknoglobals
var (
	reExtension = regexp.MustCompile(
		`  (?P<extModule>[^ ]+) (?P<extVersion>[^,]+), (?P<extImport>[^ ]+) \[(?P<extType>[^\]]+)\]`,
	)
	idxExtModule = reExtension.SubexpIndex("extModule")
	idxExtType   = reExtension.SubexpIndex("extType")
)

func withState(ctx context.Context, dir string) (context.Context, func()) {
	state := newState(dir)

	return context.WithValue(ctx, stateKey{}, state), state.cleanup
}

func getState(ctx context.Context) *state {
	v := ctx.Value(stateKey{})
	if v == nil {
		panic("no state found in context")
	}

	s, ok := v.(*state)
	if !ok {
		panic("state has wrong type")
	}

	return s
}

func newState(dir string) *state {
	return &state{dir: dir}
}

func (s *state) moduleFile() (*modfile.File, error) {
	if s._moduleFileCached != nil {
		return s._moduleFileCached, nil
	}

	filename := filepath.Join(s.dir, "go.mod")

	data, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}

	mod, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, err
	}

	s._moduleFileCached = mod

	return mod, nil
}

func (s *state) modulePath() (string, error) {
	mod, err := s.moduleFile()
	if err != nil {
		return "", err
	}

	return mod.Module.Mod.Path, nil
}

func (s *state) exePath(ctx context.Context) (string, error) {
	if len(s._exePathCached) > 0 {
		return s._exePathCached, nil
	}

	mod, err := s.moduleFile()
	if err != nil {
		return "", err
	}

	exe, err := build(ctx, mod.Module.Mod.Path, s.dir)
	if err != nil {
		return "", err
	}

	s._exePathCached = exe

	return s._exePathCached, nil
}

func (s *state) versionOutput(ctx context.Context) ([]byte, error) {
	if s._versionOutputCached != nil {
		return s._versionOutputCached, nil
	}

	exe, err := s.exePath(ctx)
	if err != nil {
		return nil, err
	}

	out, err := exec.CommandContext(ctx, exe, "version").CombinedOutput() // #nosec G204
	if err != nil {
		return nil, err
	}

	s._versionOutputCached = out

	return s._versionOutputCached, nil
}

func (s *state) hasExtension(ctx context.Context) (bool, error) {
	if s._hasExtensionCached != nil {
		return *s._hasExtensionCached, nil
	}

	out, err := s.versionOutput(ctx)
	if err != nil {
		return false, err
	}

	module, _ := parseExtensionInfo(out)

	// An extension is present if at least one module path is found in the version output.
	// The reason for this weak check is that some extensions may registered by their parent
	// extension (e.g., xk6-sql and xk6-sql-driver-* extensions), so the version output shows
	// the parent module path rather than the actual extension module path.
	// See: https://github.com/grafana/xk6/issues/327
	has := len(module) > 0

	s._hasExtensionCached = &has

	return has, nil
}

func parseExtensionInfo(line []byte) (string, string) {
	subs := reExtension.FindSubmatch(line)
	if subs == nil {
		return "", ""
	}

	return string(subs[idxExtModule]), string(subs[idxExtType])
}

func (s *state) isJS(ctx context.Context) (bool, error) {
	if s._isJSCached != nil {
		return *s._isJSCached, nil
	}

	out, err := s.versionOutput(ctx)
	if err != nil {
		return false, err
	}

	_, extType := parseExtensionInfo(out)

	js := extType == "js"

	s._isJSCached = &js

	return js, nil
}

func (s *state) cleanup() {
	if len(s._exePathCached) > 0 {
		_ = os.Remove(s._exePathCached)

		s._exePathCached = ""
	}
}
