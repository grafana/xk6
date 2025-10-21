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

func checkerVersions(_ context.Context, dir string) *checkResult {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return checkError(err)
	}

	iter, err := repo.Tags()
	if err != nil {
		return checkError(err)
	}

	const tagPrefix = "refs/tags/"

	versions := make([]*version, 0)

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		tag := strings.TrimPrefix(ref.Name().String(), tagPrefix)

		ver, err := semver.NewVersion(tag)
		if err == nil {
			versions = append(versions, &version{semver: ver, tag: tag})
		}

		return nil
	})
	if err != nil {
		return checkError(err)
	}

	if len(versions) == 0 {
		return checkFailed("missing release tags")
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[j].semver.LessThan(versions[i].semver)
	})

	return checkPassed("found `%d` versions, the latest is `%s`", len(versions), versions[0].tag)
}
