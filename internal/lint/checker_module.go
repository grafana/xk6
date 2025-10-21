package lint

import (
	"context"
)

func checkerModule(ctx context.Context, _ string) *checkResult {
	file, err := getState(ctx).moduleFile()
	if err != nil {
		return checkError(err)
	}

	return checkPassed("found `%s` as go module", file.Module.Mod.String())
}
