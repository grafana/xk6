package lint

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"golang.org/x/mod/modfile"
)

type spyKey struct{}

type spy struct {
	dir string

	_moduleFileCached    *modfile.File
	_exePathCached       string
	_versionOutputCached []byte
	_isJSCached          *bool
	_hasExtensionCached  *bool
}

func withSpy(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, spyKey{}, newSpy(dir))
}

func getSpy(ctx context.Context) *spy {
	v := ctx.Value(spyKey{})
	if v == nil {
		panic("no spy found in context")
	}

	s, ok := v.(*spy)
	if !ok {
		panic("spy has wrong type")
	}

	return s
}

func newSpy(dir string) *spy {
	return &spy{dir: dir}
}

func (s *spy) moduleFile() (*modfile.File, error) {
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

func (s *spy) modulePath() (string, error) {
	mod, err := s.moduleFile()
	if err != nil {
		return "", err
	}

	return mod.Module.Mod.Path, nil
}

func (s *spy) exePath(ctx context.Context) (string, error) {
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

func (s *spy) versionOutput(ctx context.Context) ([]byte, error) {
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

func (s *spy) hasExtension(ctx context.Context) (bool, error) {
	if s._hasExtensionCached != nil {
		return *s._hasExtensionCached, nil
	}

	mod, err := s.moduleFile()
	if err != nil {
		return false, err
	}

	out, err := s.versionOutput(ctx)
	if err != nil {
		return false, err
	}

	rex, err := regexp.Compile("(?i)  " + mod.Module.Mod.String() + "[^,]+, [^ ]+ \\[(?P<type>[a-z]+)\\]")
	if err != nil {
		return false, err
	}

	has := rex.FindAllSubmatch(out, -1) != nil

	s._hasExtensionCached = &has

	return has, nil
}

func (s *spy) isJS(ctx context.Context) (bool, error) {
	if s._isJSCached != nil {
		return *s._isJSCached, nil
	}

	mod, err := s.moduleFile()
	if err != nil {
		return false, err
	}

	out, err := s.versionOutput(ctx)
	if err != nil {
		return false, err
	}

	rex, err := regexp.Compile("(?i)  " + mod.Module.Mod.String() + "[^,]+, [^ ]+ \\[(?P<type>[a-z]+)\\]")
	if err != nil {
		return false, err
	}

	subs := rex.FindAllSubmatch(out, -1)
	if subs == nil {
		return false, nil
	}

	js := false

	for _, one := range subs {
		if string(one[rex.SubexpIndex("type")]) == "js" {
			js = true

			break
		}
	}

	s._isJSCached = &js

	return js, nil
}

func (s *spy) cleanup() {
	if len(s._exePathCached) > 0 {
		_ = os.Remove(s._exePathCached)

		s._exePathCached = ""
	}
}
