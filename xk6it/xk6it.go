// Package xk6it contains integration test runner of the xk6 extensions.
package xk6it

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"go.k6.io/xk6"
	"golang.org/x/mod/modfile"
)

// Run builds k6 in a temporary location and runs scripts matching the given pattern.
//
// For the given integration script, it searches for the containing go module
// and based on that, builds k6 binary for execution.
//
// The k6 build is done only once for a given go module.
//
// The k6 binary is placed in a temporary directory belonging to the
// call, which is automatically deleted by the go test runner at the end of the run.
//
// In practice, the execution corresponds to the "xk6 run script.js" command,
// but the xk6 binary is not necessary, because it uses xk6 as a library. In case of any error, t.Error() is called.
func Run(t *testing.T, pattern string) {
	t.Helper()

	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Error(err)
	}

	if len(files) == 0 {
		t.Errorf("No matching script for pattern %s", pattern)
	}

	k6bins := make(map[string]string)

	for _, filename := range files {
		modfile := findModule(t, filepath.Dir(filename))

		k6bin, found := k6bins[modfile]
		if !found {
			k6bin = build(t, filepath.Dir(modfile))

			k6bins[modfile] = k6bin
		}

		filename := filename

		t.Run(filename, func(t *testing.T) {
			t.Parallel()
			run(t, filename, k6bin)
		})
	}
}

func noError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Error(err)
	}
}

func findModule(t *testing.T, from string) string {
	t.Helper()

	dir, err := filepath.Abs(from)
	if err != nil {
		t.Error(err)
	}

	for len(dir) != 0 && dir[len(dir)-1] != filepath.Separator {
		file := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(file); err == nil { //nolint:forbidigo
			return file
		}
	}

	t.Errorf("go.mod file not found in %s directory or any parent directory", from)

	return ""
}

func build(t *testing.T, dir string) string {
	t.Helper()

	moduleFile := filepath.Clean(findModule(t, dir))

	bytes, err := os.ReadFile(moduleFile) //nolint:forbidigo
	noError(t, err)

	gomod, err := modfile.Parse(moduleFile, bytes, nil)
	noError(t, err)

	builder := xk6.Builder{
		Extensions: []xk6.Dependency{{PackagePath: gomod.Module.Mod.Path}},
		Replacements: []xk6.Replace{
			xk6.NewReplace(gomod.Module.Mod.Path, filepath.Dir(moduleFile)),
		},
	}

	k6bin := filepath.Join(t.TempDir(), "k6")

	if runtime.GOOS == "windows" {
		k6bin += ".exe"
	}

	noError(t, builder.Build(context.TODO(), k6bin))

	return k6bin
}

func run(t *testing.T, filename string, k6bin string) {
	t.Helper()

	args := []string{"run", "--quiet", "--no-usage-report", filename}

	cmd := exec.Command(k6bin, args...) //nolint:gosec
	cmd.Stdin = os.Stdin                //nolint:forbidigo
	cmd.Stdout = os.Stdout              //nolint:forbidigo
	cmd.Stderr = os.Stderr              //nolint:forbidigo

	noError(t, cmd.Start())
	noError(t, cmd.Wait())
}
