package lint

import (
	"context"
	"sort"

	"github.com/Masterminds/semver/v3"
	gitcmd "go.k6.io/xk6/internal/git"
)

type version struct {
	semver *semver.Version
	tag    string
}

func checkerVersions(ctx context.Context, dir string) *checkResult {
	tags, err := gitcmd.Tags(ctx, dir)
	if err != nil {
		return checkError(err)
	}

	versions := make([]*version, 0)

	for _, tag := range tags {
		ver, err := semver.NewVersion(tag)
		if err == nil {
			versions = append(versions, &version{semver: ver, tag: tag})
		}
	}

	if len(versions) == 0 {
		return checkFailed("missing release tags")
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[j].semver.LessThan(versions[i].semver)
	})

	return checkPassed("found `%d` versions, the latest is `%s`", len(versions), versions[0].tag)
}
