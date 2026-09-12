package lint

import (
	"context"

	gitcmd "go.k6.io/xk6/internal/git"
)

func checkerGit(ctx context.Context, dir string) *checkResult {
	err := gitcmd.IsWorktree(ctx, dir)
	if err != nil {
		return checkError(err)
	}

	return checkPassed("found git worktree")
}
