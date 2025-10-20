package lint

import (
	"context"
	"path/filepath"
	"regexp"
)

var reIndexDTS = regexp.MustCompile("^index.d.ts$")

func checkerTypes(ctx context.Context, dir string) *checkResult {
	s := getSpy(ctx)

	js, err := s.isJS(ctx)
	if err != nil {
		return checkError(err)
	}

	if !js {
		return checkPassed("skipped due to non JavaScript extension")
	}

	_, shortname, err := findFile(reIndexDTS,
		dir,
		filepath.Join(dir, "docs"),
		filepath.Join(dir, "api-docs"),
	)
	if err != nil {
		return checkError(err)
	}

	if len(shortname) > 0 {
		return checkPassed("found `index.d.ts` file")
	}

	return checkFailed("no `index.d.ts` file found")
}
