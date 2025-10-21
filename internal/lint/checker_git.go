package lint

import (
	"context"

	"github.com/go-git/go-git/v5"
)

func checkerGit(_ context.Context, dir string) *checkResult {
	_, err := git.PlainOpen(dir)
	if err != nil {
		return checkError(err)
	}

	return checkPassed("found git worktree")
}
