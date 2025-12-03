package test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
	"github.com/goreleaser/fileglob"
)

// k6ExitCodes maps k6 exit codes to their meanings
// Based on https://github.com/grafana/k6/blob/master/errext/exitcodes/codes.go
var k6ExitCodes = map[int]string{ //nolint:gochecknoglobals
	97:  "Cloud test run failed",
	98:  "Cloud failed to get progress",
	99:  "Thresholds have failed",
	100: "Setup timeout",
	101: "Teardown timeout",
	102: "Generic timeout",
	103: "Script stopped from REST API",
	104: "Invalid config",
	105: "External abort",
	106: "Cannot start REST API",
	107: "Script exception",
	108: "Script aborted",
	109: "Go panic",
	110: "Marked as failed",
}

func runFiles(ctx context.Context, opts *Options) ([]*ctrf.TestResult, error) {
	filenames := make([]string, 0)

	for _, pattern := range opts.Patterns {
		files, err := fileglob.Glob(pattern)
		if err != nil {
			return nil, err
		}

		filenames = append(filenames, files...)
	}

	if len(filenames) == 0 {
		return nil, ErrNoTestFiles
	}

	sort.Strings(filenames)

	filenames = slices.Compact(filenames)

	results := make([]*ctrf.TestResult, 0, len(filenames))

	for _, filename := range filenames {
		r := runFile(ctx, opts, filename)

		results = append(results, r)
	}

	return results, nil
}

func runFile(ctx context.Context, opts *Options, filename string) *ctrf.TestResult {
	start := time.Now()

	stdout := io.Discard

	args := []string{
		"--log-format=json",
		"--no-usage-report",
		"--address=127.0.0.1:0",
		"--vus=1",
		"--iterations=1",
		"--no-color",
	}

	if opts.Verbose {
		stdout = opts.Stdout

		args = append(args, "--verbose")
	} else {
		args = append(args, "--quiet", "--summary-mode=disabled")
	}

	args = append(args, "run", filename)

	cmd := exec.CommandContext( //#nosec:G204
		ctx,
		opts.K6,
		args...,
	)

	cmd.Stdout = stdout

	details, err := runAndLog(ctx, cmd, "test.file", filename)
	if err != nil {
		return errorToResult(err, details, filename)
	}

	return &ctrf.TestResult{
		Name:     filepath.Base(filename),
		Filepath: filename,
		Status:   ctrf.TestPassed,
		Duration: time.Since(start).Milliseconds(),
	}
}

func runAndLog(ctx context.Context, cmd *exec.Cmd, logargs ...any) (string, error) {
	var details string

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	decoder := json.NewDecoder(stderr)

	for {
		var entry logEntry

		err := decoder.Decode(&entry)
		if err != nil {
			if err == io.EOF {
				break
			}

			return "", fmt.Errorf("error decoding k6 log entry: %w", err)
		}

		reason := entry.log(ctx, logargs...)
		if len(details) == 0 && len(reason) > 0 {
			details = reason
		}
	}

	err = cmd.Wait()

	return details, err
}

func errorToResult(err error, details string, filename string) *ctrf.TestResult {
	message := details

	// If no details were captured, use the error message
	if len(message) == 0 {
		message = err.Error()

		var exiterr *exec.ExitError

		// Check if it's an exit error to extract the exit code
		if errors.As(err, &exiterr) {
			code := exiterr.ExitCode()

			// Map the exit code to a known k6 exit code description
			if desc, ok := k6ExitCodes[code]; ok {
				message = desc
			} else {
				message = fmt.Sprintf("k6 exited with code %d", code)
			}
		}
	}

	return &ctrf.TestResult{
		Name:     filepath.Base(filename),
		Filepath: filename,
		Status:   ctrf.TestFailed,
		Message:  message,
	}
}
