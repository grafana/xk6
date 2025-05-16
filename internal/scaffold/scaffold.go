// Package scaffold provides functionality to scaffold a new k6 extension
// based on a sample extension.
package scaffold

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Scaffold scaffolds a new k6 extension based on provided sample and actual sample parameters.
func Scaffold(ctx context.Context, sample *Sample, actual *Sample, parentDir string) error {
	dir := filepath.Join(parentDir, path.Base(actual.Module))

	repo, err := git.PlainCloneContext(ctx, dir, false, &git.CloneOptions{URL: sample.Repository})
	if err != nil {
		return err
	}

	latest, err := getLatest(repo)
	if err != nil {
		return err
	}

	if len(latest) == 0 {
		slog.Warn("No releases, the default branch will be used!", "repo", sample.Repository)
	} else {
		err = checkout(repo, latest)
		if err != nil {
			return err
		}
	}

	if err := os.RemoveAll(filepath.Join(dir, ".git")); err != nil {
		return err
	}

	return customize(dir, newReplacer(sample, actual))
}

// Adjust customizes the scaffolded extension based on the provided sample and actual sample parameters.
func Adjust(dir string, sample *Sample, actual *Sample) error {
	return customize(dir, newReplacer(sample, actual))
}

func getLatest(repo *git.Repository) (plumbing.ReferenceName, error) {
	iter, err := repo.Tags()
	if err != nil {
		return "", err
	}

	versions := make([]*semver.Version, 0)

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		ver, err := semver.NewVersion(ref.Name().Short())
		if err == nil {
			versions = append(versions, ver)
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", nil
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[j].LessThan(versions[i])
	})

	return plumbing.NewTagReferenceName(versions[0].Original()), nil
}

func checkout(repo *git.Repository, name plumbing.ReferenceName) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	return wt.Checkout(&git.CheckoutOptions{Branch: name})
}

func customize(dir string, replacer *strings.Replacer) error {
	special := []string{filepath.Join(dir, ".git"), filepath.Join(dir, "LICENSE")}

	return filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		for _, s := range special {
			if strings.HasPrefix(path, s) {
				// Skip special files
				return nil
			}
		}

		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return err
		}

		input := string(data)
		output := replacer.Replace(input)

		if input != output {
			return os.WriteFile(path, []byte(output), info.Mode())
		}

		return nil
	})
}
