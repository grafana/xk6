// Package test contains the public API of the k6 extension integration test runner.
package test

import (
	"context"
	"errors"

	"github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
)

// ErrNoTestFiles is returned when no test files match the provided patterns.
var ErrNoTestFiles = errors.New("no test files matched the provided patterns")

// Test executes the tests specified in opts and returns a CTRF report.
func Test(ctx context.Context, opts *Options) (*ctrf.Report, error) {
	results, err := runFiles(ctx, opts)
	if err != nil {
		return nil, err
	}

	r := ctrf.NewReport("", nil)

	r.Results.Tests = results

	updateSummary(r.Results.Summary, results)

	return r, nil
}

func updateSummary(summary *ctrf.Summary, results []*ctrf.TestResult) {
	summary.Tests = len(results)
	summary.Passed = 0
	summary.Failed = 0
	summary.Skipped = 0
	summary.Pending = 0
	summary.Other = 0

	for _, result := range results {
		switch result.Status {
		case ctrf.TestPassed:
			summary.Passed++
		case ctrf.TestFailed:
			summary.Failed++
		case ctrf.TestSkipped:
			summary.Skipped++
		case ctrf.TestPending:
			summary.Pending++
		case ctrf.TestOther:
			fallthrough
		default:
			summary.Other++
		}
	}
}
