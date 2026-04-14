package k6foundry

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

var (
	moduleVersionRegexp = regexp.MustCompile(`.+/v(\d+)$`)
	majorVersionRegexp  = regexp.MustCompile(`^v[0-9]+$`)

	ErrInvalidDependencyFormat = errors.New("invalid dependency format") //nolint:revive
)

const k6BaseModulePath = "go.k6.io/k6"

// k6ModulePath returns the correct Go module path for k6 given a version and an optional major override.
// For semver versions (e.g. "v2.0.0"), the major is derived from the version.
// For non-semver versions ("latest", commit SHAs), majorOverride (e.g. "v2") is used when provided.
// If neither applies, the base path "go.k6.io/k6" is returned.
func k6ModulePath(version, majorOverride string) (string, error) {
	if semver.IsValid(version) {
		return versionedPath(k6BaseModulePath, version)
	}

	if majorOverride == "" {
		return k6BaseModulePath, nil
	}

	if !majorVersionRegexp.MatchString(majorOverride) {
		return "", fmt.Errorf("%w: invalid major version %q (expected format: vN)", ErrInvalidDependencyFormat, majorOverride)
	}

	switch majorOverride {
	case "v0", "v1":
		return k6BaseModulePath, nil
	default:
		return k6BaseModulePath + "/" + majorOverride, nil
	}
}

// Module reference a go module and its version
type Module struct {
	// The name (import path) of the go module. If at a version > 1,
	// it should contain semantic import version (i.e. "/v2").
	// Used with `go get`.
	Path string

	// The version of the Go module, as used with `go get`.
	Version string

	// Module replacement
	ReplacePath string

	//  Module replace version
	ReplaceVersion string
}

func (m Module) String() string {
	version := ""
	if m.Version != "" {
		version = "@" + m.Version
	}

	replace := ""
	if m.ReplacePath != "" {
		replaceVer := ""
		if m.ReplaceVersion != "" {
			replaceVer = "@" + m.ReplaceVersion
		}
		replace = fmt.Sprintf(" => %s%s", m.ReplacePath, replaceVer)
	}

	return fmt.Sprintf("%s%s%s", m.Path, version, replace)
}

// ParseModule parses a module from a string of the form path[@version][=replace[@version]]
func ParseModule(modString string) (Module, error) {
	mod, replaceMod, _ := strings.Cut(modString, "=")

	path, version, err := splitPathVersion(mod)
	if err != nil {
		return Module{}, err
	}

	if err = checkPath(path); err != nil {
		return Module{}, err
	}

	// TODO: should we enforce the versioned path or reject if it not conformant?
	path, err = versionedPath(path, version)
	if err != nil {
		return Module{}, err
	}

	replacePath, replaceVersion, err := replace(replaceMod)
	if err != nil {
		return Module{}, err
	}

	return Module{
		Path:           path,
		Version:        version,
		ReplacePath:    replacePath,
		ReplaceVersion: replaceVersion,
	}, nil
}

// check if the path adheres to the go module path format.
// also accepts a path with only the module name
func checkPath(path string) error {
	if !strings.Contains(path, "/") {
		return nil
	}

	if err := module.CheckPath(path); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidDependencyFormat, err)
	}

	return nil
}

func replace(replaceMod string) (string, string, error) {
	if replaceMod == "" {
		return "", "", nil
	}

	replacePath, replaceVersion, err := splitPathVersion(replaceMod)
	if err != nil {
		return "", "", err
	}

	// is a relative path
	if strings.HasPrefix(replacePath, ".") {
		if replaceVersion != "" {
			return "", "", fmt.Errorf("%w: relative replace path can't specify version", ErrInvalidDependencyFormat)
		}
		return replacePath, replaceVersion, nil
	}

	return replacePath, replaceVersion, nil
}

// splits a path[@version] string into its components
func splitPathVersion(mod string) (string, string, error) {
	path, version, found := strings.Cut(mod, "@")

	// TODO: add regexp for checking path@version
	if path == "" || (found && version == "") {
		return "", "", fmt.Errorf("%w: %q", ErrInvalidDependencyFormat, mod)
	}

	return path, version, nil
}

// VersionedPath returns a module path with the major component of version added,
// if it is a valid semantic version and is > 1
// Examples:
// - Path="foo" and Version="v1.0.0" returns "foo"
// - Path="foo" and Version="v2.0.0" returns "foo/v2"
// - Path="foo/v2" and vVersion="v3.0.0" returns an error
// - Path="foo" and Version="latest" returns "foo"
func versionedPath(path string, version string) (string, error) {
	// if not is a semantic version return (could have been a commit SHA or 'latest')
	if !semver.IsValid(version) {
		return path, nil
	}
	major := semver.Major(version)

	// if the module path has a major version at the end, check for inconsistencies
	if moduleVersionRegexp.MatchString(path) {
		modPathVer := path[strings.LastIndex(path, "/")+1:]
		if modPathVer != major {
			return "", fmt.Errorf("invalid version for versioned path %q: %q", path, version)
		}
		return path, nil
	}

	// if module path does not specify major version, add it if > 1
	switch major {
	case "v0", "v1":
		return path, nil
	default:
		return path + "/" + major, nil
	}
}
