// Package lint contains the public API of the k6 extension linter.
package lint

import (
	"context"
	"time"
)

// Lint checks the directory specified in the dir parameter as the k6 extension source directory.
func Lint(ctx context.Context, dir string, opts *Options) (*Compliance, error) {
	c := new(Compliance)

	c.Timestamp = float64(time.Now().Unix())
	c.Checks, c.Passed = runChecks(ctx, dir, opts)

	return c, nil
}
