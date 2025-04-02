// Package efa enables the assignment of environment variables to go flags.
package efa

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FlagSet represents a set of defined flags for the purpose of environment variable assignment.
// Both FlagSet in the go standard library flag package and FlagSet in the spf13/pflag package extend this interface.
// FlagSet in both packages can be used as a parameter of type FlagSet.
type FlagSet interface {
	// Set sets the value of the named flag.
	Set(name, value string) error
}

// Annotatable represents a FlagSet that allows flags to be annotated.
// FlagSet in the spf13/pflag package implements the Annotatable interface.
type Annotatable interface {
	// SetAnnotation allows to set arbitrary annotations on a flag in the FlagSet.
	SetAnnotation(name, key string, values []string) error
}

// LookupFunc retrieves the value of the environment variable named by the key.
// If the variable is present in the environment the value (which may be empty) is returned and the boolean is true.
// Otherwise the returned value will be empty and the boolean will be false.
type LookupFunc func(string) (string, bool)

// Annotation name that stores the bounded environment variable names in case of annotatable FlagSet.
const Annotation = "environment"

// Efa sets flags based on environment variables and an optional variable name prefix.
type Efa struct {
	prefix   string
	lookup   LookupFunc
	set      func(name, value string) error
	annotate func(name, key string, values []string) error
}

var (
	defaultInstance *Efa //nolint:gochecknoglobals

	defaultInstanceOnce sync.Once //nolint:gochecknoglobals
)

// GetEfa returns the default Eta instance.
// The default instance will use:
// the executable name as the environment variable prefix,
// the os.LookupEnv function as the environment variable lookup function,
// and flag.CommandLine as the FlagSet.
func GetEfa() *Efa {
	defaultInstanceOnce.Do(func() {
		defaultInstance = New(flag.CommandLine, exeName(), nil)
	})

	return defaultInstance
}

func exeName() string {
	exe, err := os.Executable()
	if err != nil {
		exe = os.Args[0]
	}

	return filepath.Base(exe)
}

// New returns a new Eta instance.
// The flags FlagSet parameter is used to set the value of the flags,
// if nil then flag.CommandLine will be used instead.
// An optional environment variable name prefix can be specified in the prefix parameter.
// If the prefix parameter is not an empty string, then its value will be used as a prefix
// for the environment variable names associated with the flags (separated by the _ character).
// The lookup parameter can be specified as the environment variable lookup function,
// if its value is nil, os.LookupEnv will be used instead.
func New(flags FlagSet, prefix string, lookup LookupFunc) *Efa {
	if flags == nil {
		flags = flag.CommandLine
	}

	if lookup == nil {
		lookup = os.LookupEnv
	}

	var annotate func(name, key string, values []string) error

	if aflags, annotatable := flags.(Annotatable); annotatable {
		annotate = aflags.SetAnnotation
	}

	return &Efa{set: flags.Set, prefix: prefix, lookup: lookup, annotate: annotate}
}

const sep = "_"

func toUpperSnake(str string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(str, " ", sep), "-", sep), ".", sep))
}

// Bind assigns the corresponding environment variables to the flags named as parameters.
// The name of the environment variable is formed from the executable name as prefix and the flag name
// using screaming snake case conversion.
// Spaces and hyphens are replaced with underscores and letters are converted to uppercase.
// If the environment variable exists, the flag is set to its value.
// The flag's "environment" annotation is set to the name of the environment variable.
//
// Must be called before parsing FlagSet.
func Bind(names ...string) error {
	return GetEfa().Bind(names...)
}

// Bind assigns the corresponding environment variables to the flags named as parameters.
// The name of the environment variable is formed from the optional prefix and the flag name
// using screaming snake case conversion.
// Spaces and hyphens are replaced with underscores and letters are converted to uppercase.
// If the environment variable exists, the flag is set to its value.
// The flag's "environment" annotation is set to the name of the environment variable.
//
// Must be called before parsing FlagSet.
func (e *Efa) Bind(names ...string) error {
	prefix := e.prefix
	if len(prefix) > 0 {
		prefix += sep
	}

	for _, name := range names {
		envvar := toUpperSnake(prefix + name)

		if err := e.BindTo(name, envvar); err != nil {
			return err
		}
	}

	return nil
}

// BindTo assigns an environment variable to the flag with the given name.
// The namevar parameter list contains "flag name" - "environment variable name" pairs.
// If the environment variable exists, the flag is set to its value.
// The flag's "environment" annotation is set to the name of the environment variable.
//
// Must be called before parsing FlagSet.
func BindTo(namevar ...string) error {
	return GetEfa().BindTo(namevar...)
}

// BindTo assigns an environment variable to the flag with the given name.
// The namevar parameter list contains "flag name" - "environment variable name" pairs.
// If the environment variable exists, the flag is set to its value.
// The flag's "environment" annotation is set to the name of the environment variable.
//
// Must be called before parsing FlagSet.
func (e *Efa) BindTo(namevar ...string) error {
	if len(namevar)%2 == 1 {
		return fmt.Errorf("%w: odd number of arguments", ErrInvalidArgument)
	}

	for idx := 0; idx < len(namevar); idx += 2 {
		name, envvar := namevar[idx], namevar[idx+1]

		if e.annotate != nil {
			if err := e.annotate(name, Annotation, []string{envvar}); err != nil {
				return err
			}
		}

		value, found := e.lookup(envvar)
		if !found {
			continue
		}

		if err := e.set(name, value); err != nil {
			return err
		}
	}

	return nil
}

// ErrInvalidArgument is returned if any of the arguments are invalid.
var ErrInvalidArgument = errors.New("invalid argument")
