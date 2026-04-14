// Package k6foundry contains logic for building k6 binary
package k6foundry

import (
	"context"
	"io"
)

// WarningCode identifies the type of a non-fatal build warning.
type WarningCode string

const (
	// WarnK6VersionConflict is emitted when an extension depends on a different k6 major
	// version than the one being built. The build succeeds but the affected extensions
	// will silently register with an inactive k6 runtime and have no effect at runtime.
	WarnK6VersionConflict WarningCode = "k6_version_conflict"
)

// Warning describes a non-fatal issue detected during a build.
type Warning struct {
	Code    WarningCode
	Message string
}

// BuildInfo describes the binary
type BuildInfo struct {
	Platform    string
	ModVersions map[string]string
	Warnings    []Warning
}

func newBuildInfo(platform string) *BuildInfo {
	return &BuildInfo{
		Platform:    platform,
		ModVersions: make(map[string]string),
	}
}

// Foundry defines the interface for building a k6 binary
type Foundry interface {
	// Build returns a custom k6 binary for the given version including a set of dependencies
	// The binary is build fo the given Platform
	// The mods parameter is a list of modules to include in the build. Modules can specify its own
	// replacement (for example, a local module).
	// The replacements parameter is a list of modules to replace in the build. Allows for arbitrary
	// replacement of dependencies, used for example in development environments.
	// The buildOpts parameter is a list of additional build options to pass to the go build command.
	Build(
		ctx context.Context,
		platform Platform,
		k6Version string,
		mods []Module,
		replacements []Module,
		buildOpts []string,
		out io.Writer,
	) (*BuildInfo, error)
}
