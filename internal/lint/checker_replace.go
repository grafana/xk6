package lint

import (
	"context"
)

func checkerReplace(ctx context.Context, _ string) *checkResult {
	s := getState(ctx)

	mod, err := s.moduleFile()
	if err != nil {
		return checkError(err)
	}

	if len(mod.Replace) != 0 {
		return checkFailed("the `go.mod` file contains `replace` directive(s)")
	}

	return checkPassed("no `replace` directive in the `go.mod` file")
}
