package lint

import (
	"context"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type version struct {
	semver *semver.Version
	tag    string
}

type gitChecker struct {
	repo *git.Repository

	versions []*version
}

func newGitChecker() *gitChecker {
	return new(gitChecker)
}

func (gc *gitChecker) isWorkDir(_ context.Context, dir string) *checkResult {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return checkError(err)
	}

	gc.repo = repo

	return checkPassed("found git worktree")
}

func (gc *gitChecker) hasVersions(ctx context.Context, dir string) *checkResult {
	if gc.repo == nil {
		res := gc.isWorkDir(ctx, dir)
		if !res.passed {
			return res
		}
	}

	iter, err := gc.repo.Tags()
	if err != nil {
		return checkError(err)
	}

	const tagPrefix = "refs/tags/"

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		tag := strings.TrimPrefix(ref.Name().String(), tagPrefix)

		ver, err := semver.NewVersion(tag)
		if err == nil {
			gc.versions = append(gc.versions, &version{semver: ver, tag: tag})
		}

		return nil
	})
	if err != nil {
		return checkError(err)
	}

	if len(gc.versions) == 0 {
		return checkFailed("missing release tags")
	}

	sort.Slice(gc.versions, func(i, j int) bool {
		return gc.versions[j].semver.LessThan(gc.versions[i].semver)
	})

	return checkPassed("found `%d` versions, the latest is `%s`", len(gc.versions), gc.versions[0].tag)
}
