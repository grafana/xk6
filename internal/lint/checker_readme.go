package lint

import (
	"context"
	"regexp"
)

var reREADME = regexp.MustCompile(
	`(?i)^readme\.(?:markdown|mdown|mkdn|md|textile|rdoc|org|creole|mediawiki|wiki|rst|asciidoc|adoc|asc|pod|txt)`,
)

func checkerReadme(_ context.Context, dir string) *checkResult {
	_, name, err := findFile(reREADME, dir)
	if err != nil {
		return checkError(err)
	}

	if len(name) > 0 {
		return checkPassed("found `%s` as README file", name)
	}

	return checkFailed("no README file found")
}
