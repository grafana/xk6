package lint

import (
	"context"
)

func checkerBuild(ctx context.Context, _ string) *checkResult {
	s := getState(ctx)

	modulePath, err := s.modulePath()
	if err != nil {
		return checkError(err)
	}

	has, err := s.hasExtension(ctx)
	if err != nil {
		return checkError(err)
	}

	if !has {
		return checkFailed("extension for `%s` not found in k6", modulePath)
	}

	return checkPassed("can be built with the latest k6 version")
}
