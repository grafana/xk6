package scaffold

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

// LookupSample finds a sample based on the module path and type hint.
// If the type hint is empty, it infers the type from the module path.
func LookupSample(modulePath string, typeHint string) (*Sample, error) {
	if len(typeHint) == 0 {
		if base := path.Base(modulePath); strings.HasPrefix(base, prefixOutput) {
			typeHint = typeOutput
		} else {
			typeHint = typeJavaScript
		}
	}

	return lookupByType(typeHint)
}

// IsSample checks if the given module is a sample extension.
func IsSample(module string) (bool, *Sample) {
	var atype string

	switch module {
	case moduleJavaScript:
		atype = typeJavaScript
	case moduleOutput:
		atype = typeOutput
	default:
		return false, nil
	}

	sample, _ := lookupByType(atype)

	return true, sample
}

// Sample represents a sample extension for k6.
type Sample struct {
	// Module is the module path of the sample.
	Module string
	// Description is a short description of the sample.
	Description string
	// Package is the package name used in the sample.
	Package string
	// Repository is the URL of the sample's repository.
	Repository string
}

func (s *Sample) name() string {
	return path.Base(s.Module)
}

func (s *Sample) pkg() string {
	if len(s.Package) > 0 {
		return s.Package
	}

	return strings.ReplaceAll(
		strings.TrimPrefix(
			strings.TrimPrefix(
				s.name(),
				prefixOutput,
			),
			prefixJavaScript,
		),
		"-",
		"_",
	)
}

func newReplacer(from, to *Sample) *strings.Replacer {
	var oldnew []string

	if len(to.Description) > 0 {
		oldnew = append(oldnew, from.Description, to.Description)
	}

	oldnew = append(oldnew, from.Module, to.Module, from.name(), to.name(), from.pkg(), to.pkg())

	return strings.NewReplacer(oldnew...)
}

func lookupByType(atype string) (*Sample, error) {
	var sample *Sample

	switch strings.ToLower(atype) {
	case typeOutput:
		sample = &Sample{Module: moduleOutput, Description: descriptionOutput}
	case typeJavaScript:
		sample = &Sample{Module: moduleJavaScript, Description: descriptionJavaScript}
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownType, atype)
	}

	sample.Repository = fmt.Sprintf("https://%s.git", sample.Module)

	return sample, nil
}

// ErrUnknownType is returned when the type hint is not recognized.
// It indicates that the provided type hint does not match any known sample types.
var ErrUnknownType = errors.New("unknown type")

const (
	typeJavaScript = "javascript"
	typeOutput     = "output"

	prefixJavaScript = "xk6-"
	prefixOutput     = "xk6-output-"

	moduleJavaScript = "github.com/grafana/xk6-example"
	moduleOutput     = "github.com/grafana/xk6-output-example"

	descriptionJavaScript = "Example k6 extension"
	descriptionOutput     = "Example k6 output extension"
)
