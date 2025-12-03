package test

import "io"

// Options holds the configuration for running tests.
type Options struct {
	K6       string
	Patterns []string
	Verbose  bool
	Stdout   io.Writer
}
