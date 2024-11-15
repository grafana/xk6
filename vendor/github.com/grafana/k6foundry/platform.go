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
func NewPlatform(os, arch string) Platform {
	return Platform{OS: os, Arch: arch}
}

// ParsePlatform parses a string of the format os/arch and returns the corresponding platform
func ParsePlatform(str string) (Platform, error) {
	idx := strings.IndexRune(str, '/')
	if idx <= 0 || idx == len(str)-1 {
		return Platform{}, fmt.Errorf("%w: %s", ErrInvalidPlatform, str)
	}

	return NewPlatform(str[:idx], str[idx+1:]), nil
}

// String returns the platform in the format os/arch
func (p Platform) String() string {
	return p.OS + "/" + p.Arch
}

// Supported indicates is the given platform is supported
func (p Platform) Supported() bool {
	for _, plat := range supported {
		if plat.OS == p.OS && plat.Arch == p.Arch {
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
