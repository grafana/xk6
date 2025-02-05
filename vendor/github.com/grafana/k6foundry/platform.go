package k6foundry

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

var ErrInvalidPlatform = errors.New("invalid platform") //nolint:revive

// Platform defines a target OS and architecture for building a custom binary
type Platform struct {
	OS   string
	Arch string
}

// RuntimePlatform returns the Platform of the current executable
func RuntimePlatform() Platform {
	return Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}
}

// NewPlatform creates a new Platform given the os and arch
func NewPlatform(os, arch string) (Platform, error) {
	if !isSupported(os, arch) {
		return Platform{}, fmt.Errorf("%w: %s/%s", ErrInvalidPlatform, os, arch)
	}
	return Platform{OS: os, Arch: arch}, nil
}

// ParsePlatform parses a string of the format os/arch and returns the corresponding platform
func ParsePlatform(str string) (Platform, error) {
	os, arch, found := strings.Cut(str, "/")
	if !found || os == "" || arch == "" {
		return Platform{}, fmt.Errorf("%w: %s", ErrInvalidPlatform, str)
	}

	return NewPlatform(os, arch)
}

// String returns the platform in the format os/arch
func (p Platform) String() string {
	return p.OS + "/" + p.Arch
}

// isSupported indicates is the given platform is supported
func isSupported(os string, arch string) bool {
	for _, plat := range supported {
		if os == plat.OS && arch == plat.Arch {
			return true
		}
	}

	return false
}

// SupportedPlatforms returns a list of supported platforms
func SupportedPlatforms() []Platform {
	return supported
}

var supported = []Platform{ //nolint:gochecknoglobals
	{OS: "linux", Arch: "amd64"},
	{OS: "linux", Arch: "arm64"},
	{OS: "windows", Arch: "amd64"},
	{OS: "windows", Arch: "arm64"},
	{OS: "darwin", Arch: "amd64"},
	{OS: "darwin", Arch: "arm64"},
}
