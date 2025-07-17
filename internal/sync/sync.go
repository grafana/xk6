// Package sync contains the public API of the k6 dependency synchronizer.
package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

const (
	defaultGoProxy = "https://proxy.golang.org"
	k6Module       = "go.k6.io/k6"
	modFile        = "go.mod"
	latest         = "latest"
)

var errHTTP = errors.New("HTTP error")

// Change represents a change in a module dependency.
type Change struct {
	// Module is the module path.
	Module string `json:"module,omitempty"`
	// From is the version being replaced.
	From string `json:"from,omitempty"`
	// To is the version being replaced with.
	To string `json:"to,omitempty"`
}

// Result represents the result of a synchronization operation.
type Result struct {
	// The k6 version used for synchronization.
	K6Version string `json:"k6_version,omitempty"`
	// Changes is a list of changes made to the module dependencies.
	Changes []*Change `json:"changes,omitempty"`
}

// Sync synchronizes the versions of the module dependencies in the specified directory with k6.
func Sync(ctx context.Context, dir string, opts *Options) (*Result, error) {
	slog.Debug("Syncing dependencies with k6")

	extModfile, err := loadModfile(dir)
	if err != nil {
		return nil, err
	}

	k6Version, err := getK6Version(ctx, opts, extModfile)
	if err != nil {
		return nil, err
	}

	slog.Debug("Target k6", "version", k6Version)

	k6Modfile, err := getModule(ctx, k6Module, k6Version)
	if err != nil {
		return nil, err
	}

	result := &Result{
		K6Version: k6Version,
		Changes:   diffRequires(extModfile, k6Modfile),
	}

	if len(result.Changes) == 0 {
		slog.Debug("No changes needed")

		return result, nil
	}

	if opts.DryRun {
		slog.Debug("Not saving changes, dry run")

		return result, nil
	}

	patch := make([]string, 0, len(result.Changes)+1) // +1 for the "get" command

	// Prepare the patch command to update go.mod
	patch = append(patch, "get")

	// Add each change to the patch command
	for _, change := range result.Changes {
		slog.Debug("Updating dependency", "module", change.Module, "from", change.From, "to", change.To)
		patch = append(patch, fmt.Sprintf("%s@%s", change.Module, change.To))
	}

	slog.Debug("Updating go.mod")

	cmd := exec.Command("go", patch...) // #nosec G204

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetLatestK6Version retrieves the latest version of k6 from the Go proxy.
func GetLatestK6Version(ctx context.Context) (string, error) {
	return getLatestVersion(ctx, k6Module)
}

func diffRequires(extModfile, k6Modfile *modfile.File) []*Change {
	changes := make([]*Change, 0)

	for _, k6Require := range k6Modfile.Require {
		k6Modpath, k6Modversion := k6Require.Mod.Path, k6Require.Mod.Version

		for _, extRequire := range extModfile.Require {
			extModpath, extModversion := extRequire.Mod.Path, extRequire.Mod.Version
			if k6Modpath == extModpath && k6Modversion != extModversion {
				changes = append(changes, &Change{
					Module: k6Modpath,
					From:   extModversion,
					To:     k6Modversion,
				})
			}
		}
	}

	return changes
}

func getK6Version(ctx context.Context, opts *Options, mf *modfile.File) (string, error) {
	k6Version := opts.K6Version
	if len(k6Version) == 0 {
		var found bool

		k6Version, found = findRequire(mf, k6Module)
		if !found {
			slog.Info("k6 not found in go.mod, using latest version")

			k6Version = latest
		}
	}

	if k6Version == latest {
		return getLatestVersion(ctx, k6Module)
	}

	return k6Version, nil
}

func loadModfile(dir string) (*modfile.File, error) {
	filename := filepath.Join(dir, modFile)

	data, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}

	file, err := modfile.Parse(filename, data, nil)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func findRequire(mf *modfile.File, module string) (string, bool) {
	for _, r := range mf.Require {
		if r.Mod.Path == module {
			return r.Mod.Version, true
		}
	}

	return "", false
}

func goproxy() string {
	proxy := os.Getenv("GOPROXY")
	if len(proxy) == 0 {
		return defaultGoProxy
	}

	return proxy
}

func getModule(ctx context.Context, pkg string, version string) (*modfile.File, error) {
	resp, err := goProxyGet(ctx, fmt.Sprintf("/%s/@v/%s.mod", pkg, version))
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	mf, err := modfile.Parse(modFile, data, nil)
	if err != nil {
		return nil, err
	}

	return mf, nil
}

func getLatestVersion(ctx context.Context, pkg string) (string, error) {
	resp, err := goProxyGet(ctx, fmt.Sprintf("/%s/@latest", pkg))
	if err != nil {
		return "", err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	latest := struct {
		Version string `json:"version"`
	}{}

	err = json.NewDecoder(resp.Body).Decode(&latest)
	if err != nil {
		return "", err
	}

	return latest.Version, nil
}

func goProxyGet(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, goproxy()+url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s, url: %s", errHTTP, resp.Status, url)
	}

	return resp, nil
}
