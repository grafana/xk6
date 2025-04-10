package lint

import (
	"context"
	"path/filepath"
	"regexp"
)

var reCODEOWNERS = regexp.MustCompile("^CODEOWNERS$")

func checkerCodeowners(_ context.Context, dir string) *checkResult {
	_, shortname, err := findFile(reCODEOWNERS,
		dir,
		filepath.Join(dir, ".github"),
		filepath.Join(dir, "docs"),
	)
	if err != nil {
		return checkError(err)
	}

	if len(shortname) > 0 {
		return checkPassed("found `CODEOWNERS` file")
	}

	return checkFailed("no CODEOWNERS file found")
}
