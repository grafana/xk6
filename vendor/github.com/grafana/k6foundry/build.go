// Package k6foundry contains logic for building k6 binary
package k6foundry

import (
	"context"
	"io"
)

// BuildInfo describes the binary
type BuildInfo struct {
	Platform    string
	ModVersions map[string]string
}

// Builder defines the interface for building a k6 binary
type Builder interface {
	// Build returns a custom k6 binary for the given version including a set of dependencies
	Build(
		ctx context.Context,
		platform Platform,
		k6Version string,
		mods []Module,
		buildOpts []string,
		out io.Writer,
	) (*BuildInfo, error)
}
