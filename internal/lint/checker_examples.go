package lint

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

func checkerExamples(ctx context.Context, dir string) *checkResult {
	s := getSpy(ctx)

	js, err := s.isJS(ctx)
	if err != nil {
		return checkError(err)
	}

	if !js {
		return checkPassed("skipped due to non JavaScript extension")
	}

	dir = filepath.Join(dir, "examples")

	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return checkFailed("missing `examples` directory")
		}

		return checkError(err)
	}

	if !info.IsDir() {
		return checkFailed("`examples` is not a directory")
	}

	hasRegular := false

	err = filepath.WalkDir(dir, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.Type().IsRegular() {
			hasRegular = true
		}

		return nil
	})
	if err != nil {
		return checkError(err)
	}

	if hasRegular {
		return checkPassed("found `examples` as examples directory")
	}

	return checkFailed("no examples found in the `examples` directory")
}
