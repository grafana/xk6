package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	giturls "github.com/chainguard-dev/git-urls"
	"github.com/go-git/go-git/v5"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

func getModulePath(dir string) (string, error) {
	filename := filepath.Join(dir, "go.mod")

	content, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return "", err
	}

	mod, err := modfile.Parse(filename, content, nil)
	if err != nil {
		return "", err
	}

	return mod.Module.Mod.Path, nil
}

func getGitURL(dir string) (string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		return "", err
	}

	if len(remote.Config().URLs) == 0 {
		return "", git.ErrMissingURL
	}

	return remote.Config().URLs[0], nil
}

func parseGitURL(gitURL string) (string, string, string, error) {
	u, err := giturls.Parse(gitURL)
	if err != nil {
		return "", "", "", err
	}

	name := strings.TrimSuffix(path.Base(u.Path), ".git")
	owner := path.Base(path.Dir(u.Path))

	return u.Hostname(), owner, name, nil
}

func gitURLToModule(gitURL string) (string, error) {
	hostname, owner, name, err := parseGitURL(gitURL)
	if err != nil {
		return "", err
	}

	return path.Join(hostname, owner, name), nil
}

func moduleToGitURL(modulePath string) string {
	prefix, _, _ := module.SplitPathVersion(modulePath)

	return "https://" + prefix + ".git"
}

func getDescription(ctx context.Context, gitURL string) (string, error) {
	hostname, owner, name, err := parseGitURL(gitURL)
	if err != nil {
		return "", err
	}

	// Only support GitHub for now
	if hostname != "github.com" {
		return "", nil
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+owner+"/"+name, nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}

	defer res.Body.Close() //nolint:errcheck

	if res.StatusCode != http.StatusOK {
		return "", os.ErrNotExist
	}

	var body map[string]any

	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return "", err
	}

	if desc, ok := body["description"].(string); ok {
		return desc, nil
	}

	return "", nil
}
