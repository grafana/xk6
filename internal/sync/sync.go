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
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

const (
	defaultGoProxy = "https://proxy.golang.org"
	k6BaseModule   = "go.k6.io/k6"
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

	k6ModulePath, k6Version, err := resolveK6Module(ctx, opts, extModfile)
	if err != nil {
		return nil, err
	}

	slog.Debug("Target k6", "module", k6ModulePath, "version", k6Version)

	k6Modfile, err := getModule(ctx, k6ModulePath, k6Version)
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

	cmd := exec.CommandContext(ctx, "go", patch...) // #nosec G204

	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr

	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetLatestK6Version retrieves the latest version of k6 (v1) from the Go proxy.
func GetLatestK6Version(ctx context.Context) (string, error) {
	return getLatestVersion(ctx, k6BaseModule)
}

// GetLatestK6VersionFor retrieves the latest version of the given k6 module path from the Go proxy.
// Use this for k6 major versions beyond v1 (e.g. "go.k6.io/k6/v2").
func GetLatestK6VersionFor(ctx context.Context, modulePath string) (string, error) {
	return getLatestVersion(ctx, modulePath)
}

// GetOverallLatestK6Version returns the module path and version of the highest
// published k6 release across all major versions. It starts with go.k6.io/k6
// and probes go.k6.io/k6/v2, go.k6.io/k6/v3, ... until a major is not found.
func GetOverallLatestK6Version(ctx context.Context) (modulePath, version string, err error) {
	return getOverallLatestK6Version(ctx)
}

// ResolveK6ModuleForVersion determines the k6 module path for an explicit version
// string (semver tag, pseudo-version, SHA, or branch name).
//
// For clean release tags (e.g. "v2.0.0") the major version is inferred directly.
// For everything else (SHAs, pseudo-versions, branch names) the Go proxy is
// queried: the declared module path inside the fetched go.mod is returned, so
// a SHA that belongs to a v2 commit will correctly yield "go.k6.io/k6/v2".
func ResolveK6ModuleForVersion(ctx context.Context, version string) (string, error) {
	// Clean release tags: derive the module path from the major version suffix.
	if semver.IsValid(version) && semver.Prerelease(version) == "" {
		path := k6ModulePathForSemver(version)
		slog.Debug("Inferred k6 module path from semver", "version", version, "path", path)

		return path, nil
	}

	slog.Debug("Non-semver version, probing Go proxy to find k6 module path", "version", version)

	return probeK6ModuleForVersion(ctx, version)
}

func k6ModulePathForSemver(version string) string {
	major := semver.Major(version) // e.g. "v0", "v1", "v2"
	if major == "v0" || major == "v1" {
		return k6BaseModule
	}

	return k6BaseModule + "/" + major
}

// versionInfo is the JSON response from the Go proxy /@v/<version>.info endpoint.
type versionInfo struct {
	Version string `json:"Version"` // canonical version (semver or pseudo-version)
	Time    string `json:"Time"`    // RFC 3339 commit timestamp
}

// getVersionInfo resolves an arbitrary version reference (SHA, branch name,
// semver tag, or pseudo-version) to a canonical version via the proxy's
// /@v/<version>.info endpoint. The .info endpoint is the only proxy endpoint
// that accepts non-canonical version strings such as raw commit SHAs.
func getVersionInfo(ctx context.Context, pkg, version string) (*versionInfo, error) {
	resp, err := goProxyGet(ctx, fmt.Sprintf("/%s/@v/%s.info", pkg, version))
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var info versionInfo

	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &info, nil
}

// probeK6ModuleForVersion asks the Go proxy which k6 major-version module
// contains the given version string (SHA, branch name, or pseudo-version).
//
// Per the Go module proxy protocol, only the .info endpoint accepts arbitrary
// version references such as commit SHAs. The .mod endpoint requires a
// canonical version (semver tag or pseudo-version). So we do a two-step
// lookup: .info first to resolve the canonical version, then .mod to read the
// module declaration and determine the correct major-version module path.
//
// The base module (go.k6.io/k6) is tried first because the proxy can serve
// a go.mod for a v2 commit under the v1 module path, and the module
// declaration inside that go.mod is authoritative. If the base module returns
// a 404 for the given reference, v2, v3, … are probed in order.
func probeK6ModuleForVersion(ctx context.Context, version string) (string, error) {
	slog.Debug("Non-canonical version, resolving via .info then .mod", "version", version)

	if path, ok := probeModuleForVersion(ctx, k6BaseModule, version); ok {
		return path, nil
	}

	slog.Debug("Base module probe failed, trying major versions")

	for major := 2; ; major++ {
		modPath := fmt.Sprintf("%s/v%d", k6BaseModule, major)

		path, ok := probeModuleForVersion(ctx, modPath, version)
		if !ok {
			// Probe failed; subsequent major versions won't have this reference either.
			break
		}

		// SA4004: intentional — if v2 has the reference we return; if not we break.
		// Higher majors share the same VCS history so won't have it either.
		return path, nil //nolint:staticcheck
	}

	return "", fmt.Errorf("could not find k6 major version for version %q", version)
}

// probeModuleForVersion resolves version against modPath using .info then .mod.
// Returns the declared module path and true on success, or "", false on any error.
func probeModuleForVersion(ctx context.Context, modPath, version string) (string, bool) {
	slog.Debug("Probing module for version", "module", modPath, "version", version)

	info, err := getVersionInfo(ctx, modPath, version)
	if err != nil {
		slog.Debug("Module .info probe failed", "module", modPath, "version", version, "error", err)

		return "", false
	}

	slog.Debug("Resolved canonical version via .info", "module", modPath, "canonical", info.Version)

	mf, err := getModule(ctx, modPath, info.Version)
	if err != nil || mf.Module == nil {
		slog.Debug("Module .mod fetch failed", "module", modPath, "version", info.Version, "error", err)

		return "", false
	}

	slog.Debug("Resolved k6 module path from go.mod declaration", "declared", mf.Module.Mod.Path)

	return mf.Module.Mod.Path, true
}

// ExtensionModule describes a k6 extension for the purpose of k6 module resolution.
type ExtensionModule struct {
	// Path is the Go module path of the extension (e.g. "github.com/grafana/xk6-sql").
	Path string
	// Version is the requested version; empty means resolve the latest.
	Version string
	// LocalPath is a local directory that replaces the module. When set, the
	// go.mod is read from there instead of being fetched from the proxy.
	LocalPath string
}

// ResolveK6ModuleForExtensions determines which k6 module path and version to
// use based on the dependencies declared by the given extensions. It reads each
// extension's go.mod (locally when LocalPath is set, otherwise from the Go
// proxy) and returns the first k6 require it finds. If none of the extensions
// declare k6 as a dependency, it falls back to GetOverallLatestK6Version.
func ResolveK6ModuleForExtensions(
	ctx context.Context, extensions []ExtensionModule,
) (modulePath, version string, err error) {
	slog.Debug("Resolving k6 module from extension dependencies", "count", len(extensions))

	for _, ext := range extensions {
		slog.Debug("Checking extension go.mod for k6 dependency", "module", ext.Path)

		mf, readErr := resolveExtensionModfile(ctx, ext)
		if readErr != nil {
			slog.Debug("Could not read go.mod for extension, skipping", "module", ext.Path, "error", readErr)
			continue
		}

		if k6path, k6ver, found := findK6Require(mf); found {
			slog.Debug("Found k6 dependency in extension", "extension", ext.Path, "k6module", k6path, "k6version", k6ver)

			return k6path, k6ver, nil
		}

		slog.Debug("Extension does not declare k6 as a dependency", "module", ext.Path)
	}

	slog.Debug("No extension declared k6, falling back to overall latest")

	return getOverallLatestK6Version(ctx)
}

func resolveExtensionModfile(ctx context.Context, ext ExtensionModule) (*modfile.File, error) {
	if ext.LocalPath != "" {
		slog.Debug("Reading go.mod from local path", "module", ext.Path, "local", ext.LocalPath)

		return loadModfile(ext.LocalPath)
	}

	ver := ext.Version
	if ver == "" {
		slog.Debug("No version specified for extension, resolving latest", "module", ext.Path)

		var err error

		ver, err = getLatestVersion(ctx, ext.Path)
		if err != nil {
			return nil, err
		}

		slog.Debug("Resolved latest version for extension", "module", ext.Path, "version", ver)
	}

	return getModule(ctx, ext.Path, ver)
}

func getOverallLatestK6Version(ctx context.Context) (modulePath, version string, err error) {
	bestPath := k6BaseModule

	bestVersion, err := getLatestVersion(ctx, k6BaseModule)
	if err != nil {
		return "", "", err
	}

	for major := 2; ; major++ {
		modPath := fmt.Sprintf("%s/v%d", k6BaseModule, major)

		ver, probeErr := getLatestVersion(ctx, modPath)
		if probeErr != nil {
			// This major does not exist yet; stop probing.
			break
		}

		if semver.Compare(ver, bestVersion) > 0 {
			bestPath = modPath
			bestVersion = ver
		}
	}

	// probeErr signals the major version does not exist; this is expected, not an error condition.
	return bestPath, bestVersion, nil //nolint:nilerr
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

// resolveK6Module determines the k6 module path and version to sync against.
// It checks the extension's go.mod for any k6 major version (go.k6.io/k6, go.k6.io/k6/v2, etc.)
// and falls back to opts.K6Version if set, or the overall latest version otherwise.
func resolveK6Module(ctx context.Context, opts *Options, mf *modfile.File) (modulePath, version string, err error) {
	if len(opts.K6Version) > 0 {
		return k6BaseModule, opts.K6Version, nil
	}

	if path, ver, found := findK6Require(mf); found {
		return path, ver, nil
	}

	slog.Info("k6 not found in go.mod, using overall latest version")

	return getOverallLatestK6Version(ctx)
}

// findK6Require finds a k6 module (any major version) in the given modfile.
// It matches go.k6.io/k6 as well as go.k6.io/k6/v2, go.k6.io/k6/v3, etc.
func findK6Require(mf *modfile.File) (path, version string, found bool) {
	for _, r := range mf.Require {
		if r.Mod.Path == k6BaseModule || strings.HasPrefix(r.Mod.Path, k6BaseModule+"/v") {
			return r.Mod.Path, r.Mod.Version, true
		}
	}

	return "", "", false
}

func getK6Version(ctx context.Context, opts *Options, mf *modfile.File) (string, error) {
	k6Version := opts.K6Version
	if len(k6Version) == 0 {
		var found bool

		_, k6Version, found = findK6Require(mf)
		if !found {
			slog.Info("k6 not found in go.mod, using latest version")

			k6Version = latest
		}
	}

	if k6Version == latest {
		return getLatestVersion(ctx, k6BaseModule)
	}

	return k6Version, nil
}

func loadModfile(dir string) (*modfile.File, error) {
	filename := filepath.Join(dir, modFile)

	data, err := os.ReadFile(filepath.Clean(filename)) //nolint:forbidigo
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
	proxy := os.Getenv("GOPROXY") //nolint:forbidigo
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
	fullURL := goproxy() + url

	slog.Debug("Go proxy request", "url", fullURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req) //nolint:gosec
	if err != nil {
		slog.Debug("Go proxy request failed", "url", fullURL, "error", err)

		return nil, err
	}

	//nolint:gosec // URL is constructed from GOPROXY env var and internal paths, not user input
	slog.Debug("Go proxy response", "url", fullURL, "status", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s, url: %s", errHTTP, resp.Status, url)
	}

	return resp, nil
}
