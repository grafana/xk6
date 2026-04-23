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
	"time"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

const (
	defaultGoProxy = "https://proxy.golang.org"
	k6BaseModule   = "go.k6.io/k6"
	modFile        = "go.mod"
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

// GetLatestVersion retrieves the latest version of the given module path from the Go proxy.
func GetLatestVersion(ctx context.Context, modulePath string) (string, error) {
	return getLatestVersion(ctx, modulePath)
}

// GetOverallLatestVersionFor returns the module path and version of the highest
// published release of baseModule across all major versions. It probes baseModule,
// baseModule/v2, baseModule/v3, … until a major is not found.
func GetOverallLatestVersionFor(ctx context.Context, baseModule string) (modulePath, version string, err error) {
	return getOverallLatestVersionFor(ctx, baseModule)
}

// ResolveModuleForVersion determines the versioned module path for baseModule at
// the given version string (semver tag, pseudo-version, SHA, or branch name).
//
// For clean release tags (e.g. "v2.0.0") the major version is inferred directly.
// For everything else the Go proxy is queried: starting from baseModule it probes
// baseModule/v2, baseModule/v3, … until the version is found. This allows callers
// to pass a custom fork base path (e.g. "github.com/myfork/k6") and get back the
// correctly versioned path (e.g. "github.com/myfork/k6/v2") without having to
// know the major version in advance.
func ResolveModuleForVersion(ctx context.Context, baseModule, version string) (string, error) {
	if semver.IsValid(version) && semver.Prerelease(version) == "" {
		major := semver.Major(version)

		var path string
		if major == "v0" || major == "v1" {
			path = baseModule
		} else {
			path = baseModule + "/" + major
		}

		slog.Debug("Inferred module path from semver", "base", baseModule, "version", version, "path", path)

		return path, nil
	}

	slog.Debug("Non-semver version, probing Go proxy to find module path", "base", baseModule, "version", version)

	return probeModuleVersionForBase(ctx, baseModule, version)
}

// versionInfo is the JSON response from the Go proxy /@v/<version>.info endpoint.
type versionInfo struct {
	Version string `json:"Version"` // canonical version (semver or pseudo-version)
	Time    string `json:"Time"`    // RFC 3339 commit timestamp
}

// probeVersionInfo calls /@v/<version>.info for pkg.
// On 200 it returns the canonical version string. Any non-200 response is an error.
func probeVersionInfo(ctx context.Context, pkg, version string) (string, error) {
	path, err := proxyPath(pkg, fmt.Sprintf("/@v/%s.info", version))
	if err != nil {
		return "", err
	}

	resp, err := goProxyGet(ctx, path)
	if err != nil {
		return "", err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("%w: %s, url: /%s/@v/%s.info", errHTTP, resp.Status, pkg, version)
	}

	var info versionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", err
	}

	return info.Version, nil
}

// probeModuleVersionForBase asks the Go proxy which major-version of the given
// base module contains the version string (SHA, branch name, or pseudo-version).
// It first probes the base module, then iterates v2, v3, … until found or until
// two consecutive major versions do not exist in the registry.
func probeModuleVersionForBase(ctx context.Context, baseModule, version string) (string, error) {
	slog.Debug("Non-canonical version, resolving via .info", "version", version)

	_, baseErr := probeVersionInfo(ctx, baseModule, version)
	if baseErr == nil {
		// The proxy enforces that module path in go.mod matches the requested path,
		// so a 200 here guarantees this commit belongs to baseModule.
		return baseModule, nil
	}

	slog.Debug("Base module .info probe failed, falling through to loop", "base", baseModule, "error", baseErr)

	// Fallback loop: iterate v2, v3, … when the base module does not contain the version.
	slog.Debug("Iterating major versions", "base", baseModule)

	const maxConsecutiveAbsent = 2

	consecutiveAbsent := 0

	for major := 2; ; major++ {
		modPath := fmt.Sprintf("%s/v%d", baseModule, major)

		if path, ok := probeModuleForVersion(ctx, modPath, version); ok {
			return path, nil
		}

		if _, err := getLatestVersion(ctx, modPath); err != nil {
			consecutiveAbsent++
			slog.Debug("Major version does not exist", "module", modPath, "consecutiveAbsent", consecutiveAbsent)

			if consecutiveAbsent >= maxConsecutiveAbsent {
				slog.Debug("Stopping probe after consecutive absent majors", "base", baseModule)
				break
			}
		} else {
			consecutiveAbsent = 0
			slog.Debug("Major version exists but does not contain SHA, trying next", "module", modPath)
		}
	}

	return "", fmt.Errorf("could not find major version module for %q at version %q", baseModule, version)
}

// probeModuleForVersion is used by the fallback loop in probeModuleVersionForBase.
// Returns the module path and true if the version is found under modPath, false otherwise.
func probeModuleForVersion(ctx context.Context, modPath, version string) (string, bool) {
	slog.Debug("Probing module for version", "module", modPath, "version", version)

	canonical, err := probeVersionInfo(ctx, modPath, version)
	if err != nil {
		slog.Debug("Module .info probe failed", "module", modPath, "version", version, "error", err)
		return "", false
	}

	slog.Debug("Resolved canonical version via .info", "module", modPath, "canonical", canonical)

	return modPath, true
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

	return getOverallLatestVersionFor(ctx, k6BaseModule)
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

func getOverallLatestVersionFor(ctx context.Context, baseModule string) (modulePath, version string, err error) {
	bestVersion, err := getLatestVersion(ctx, baseModule)
	if err != nil {
		return "", "", err
	}

	bestPath := baseModule

	for major := 2; ; major++ {
		modPath := fmt.Sprintf("%s/v%d", baseModule, major)

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
		path, err := ResolveModuleForVersion(ctx, k6BaseModule, opts.K6Version)
		if err != nil {
			return "", "", err
		}

		return path, opts.K6Version, nil
	}

	if path, ver, found := findK6Require(mf); found {
		return path, ver, nil
	}

	slog.Info("k6 not found in go.mod, using overall latest version")

	return getOverallLatestVersionFor(ctx, k6BaseModule)
}

// findK6Require finds a k6 module (any major version) in the given modfile.
// It matches go.k6.io/k6 as well as go.k6.io/k6/v2, go.k6.io/k6/v3, etc.
func findK6Require(mf *modfile.File) (path, version string, found bool) {
	for _, r := range mf.Require {
		base, _, _ := module.SplitPathVersion(r.Mod.Path)
		if base == k6BaseModule {
			return r.Mod.Path, r.Mod.Version, true
		}
	}

	return "", "", false
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

// proxyPath builds a proxy URL path with the module path and version properly
// escaped per the module proxy protocol (uppercase letters encoded as !lowercase).
func proxyPath(modulePath, suffix string) (string, error) {
	escaped, err := module.EscapePath(modulePath)
	if err != nil {
		return "", fmt.Errorf("invalid module path %q: %w", modulePath, err)
	}

	return "/" + escaped + suffix, nil
}

func goproxy() string {
	proxy := os.Getenv("GOPROXY") //nolint:forbidigo
	if len(proxy) == 0 {
		return defaultGoProxy
	}

	return proxy
}

func getModule(ctx context.Context, pkg string, version string) (*modfile.File, error) {
	path, err := proxyPath(pkg, fmt.Sprintf("/@v/%s.mod", version))
	if err != nil {
		return nil, err
	}

	resp, err := goProxyGet(ctx, path)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s, url: /%s/@v/%s.mod", errHTTP, resp.Status, pkg, version)
	}

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
	path, err := proxyPath(pkg, "/@latest")
	if err != nil {
		return "", err
	}

	resp, err := goProxyGet(ctx, path)
	if err != nil {
		return "", err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: %s, url: /%s/@latest", errHTTP, resp.Status, pkg)
	}

	latest := struct {
		Version string `json:"version"`
	}{}

	err = json.NewDecoder(resp.Body).Decode(&latest)
	if err != nil {
		return "", err
	}

	return latest.Version, nil
}

// goProxyGet fetches a URL from the Go module proxy and returns the response.
// It retries on network errors and 5xx responses (up to 3 attempts, with 1s/2s backoff).
// Responses with non-5xx status codes (including 404) are returned as-is; callers
// are responsible for checking resp.StatusCode and closing resp.Body.
func goProxyGet(ctx context.Context, url string) (*http.Response, error) {
	fullURL := goproxy() + url

	slog.Debug("Go proxy request", "url", fullURL)

	const maxAttempts = 3

	var lastErr error

	for attempt := range maxAttempts {
		if attempt > 0 {
			delay := time.Duration(1<<(attempt-1)) * time.Second // 1s, 2s
			t := time.NewTimer(delay)
			select {
			case <-t.C:
			case <-ctx.Done():
				t.Stop()
				return nil, ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
		if err != nil {
			return nil, err // malformed request, no point retrying
		}

		resp, err := http.DefaultClient.Do(req) //nolint:gosec
		if err != nil {
			slog.Debug("Go proxy request failed", "url", fullURL, "attempt", attempt+1, "error", err)
			lastErr = err

			continue
		}

		slog.Debug("Go proxy response", "url", fullURL, "status", resp.StatusCode) //nolint:gosec

		// Retry on server-side errors; return everything else to the caller for status checking.
		if resp.StatusCode >= 500 {
			slog.Debug("Go proxy server error, will retry", //nolint:gosec
				"url", fullURL, "attempt", attempt+1, "status", resp.StatusCode)
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("%w: %s, url: %s", errHTTP, resp.Status, url)

			continue
		}

		return resp, nil
	}

	return nil, lastErr
}
