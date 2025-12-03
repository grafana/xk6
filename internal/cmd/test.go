package cmd

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"github.com/spf13/cobra"
	"github.com/szkiba/efa"
	"go.k6.io/xk6/internal/test"
)

type testOptions struct {
	*buildOptions

	k6      string
	verbose bool
	out     string
	json    bool
	compact bool

	stdout io.Writer
}

func newTestOptions() *testOptions {
	opts := &testOptions{
		buildOptions: newBuildOptions(),
		k6:           "",
	}

	return opts
}

const exitCodeTestFailed = 2

//go:embed help/test.md
var testHelp string

var errTestFailed = errors.New("test failed")

func testCmd() *cobra.Command {
	opts := newTestOptions()

	cmd := &cobra.Command{
		Use:   "test [flags] pattern",
		Short: shortHelp(testHelp),
		Long:  testHelp,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Lookup("verbose").Changed {
				opts.verbose = true
			}

			opts.stdout = cmd.OutOrStdout()

			err := runTestE(cmd.Context(), opts, args)
			if errors.Is(err, errTestFailed) {
				slog.Error(errTestFailed.Error())
				os.Exit(exitCodeTestFailed)
			}

			return err
		},
		DisableAutoGenTag: true,
	}

	flags := cmd.Flags()

	flags.SortFlags = false

	cobra.CheckErr(buildCommonFlags(flags, opts.buildOptions))

	flags.StringVar(&opts.k6, "k6", "", "Specify the k6 binary to use instead of building one")
	flags.StringVarP(&opts.out, "out", "o", "", "Write output to file instead of stdout")
	flags.BoolVar(&opts.json, "json", false, "Generate JSON output")
	flags.BoolVarP(&opts.compact, "compact", "c", false, "Compact instead of pretty-printed JSON output")

	env := efa.New(flags, "", nil)

	cobra.CheckErr(env.Bind("k6"))

	return cmd
}

func runTestE(ctx context.Context, opts *testOptions, args []string) (result error) {
	output := colorable.NewColorableStdout()

	if len(opts.out) > 0 {
		file, err := os.Create(opts.out)
		if err != nil {
			return err
		}

		defer func() {
			err := file.Close()
			if result == nil && err != nil {
				result = err
			}
		}()

		output = colorable.NewNonColorable(file)
	}

	if len(opts.k6) == 0 {
		cleanup, err := buildK6OnTheFly(ctx, opts.buildOptions)
		if err != nil {
			return err
		}

		defer cleanup()

		opts.k6 = opts.output
	}

	topts := &test.Options{
		K6:       opts.k6,
		Patterns: args,
		Verbose:  opts.verbose,
		Stdout:   opts.stdout,
	}

	report, err := test.Test(ctx, topts)
	if err != nil {
		return err
	}

	report.GeneratedBy = appname
	report.Results.Tool = &ctrf.Tool{
		Name:    appname,
		Version: getVersion(),
	}

	testLogSummary(report.Results.Summary)

	err = testOutput(opts.json, report, output, opts.compact)
	if err != nil {
		return err
	}

	if report.Results.Summary.Failed > 0 {
		return errTestFailed
	}

	return nil
}

func testLogSummary(summary *ctrf.Summary) {
	if summary.Tests == 0 {
		slog.Warn("No test files matched the given patterns")
	}

	if summary.Failed > 0 {
		slog.Error("Some tests failed", "failed", summary.Failed, "total", summary.Tests)
	} else {
		slog.Info("All tests passed", "total", summary.Tests)
	}
}

func testOutput(json bool, report *ctrf.Report, output io.Writer, compact bool) error {
	if json {
		return testJSONOutput(report, output, compact)
	}

	return testTextOutput(report, output)
}

func formatDetails(details string) string {
	if !strings.Contains(details, "\n") {
		return details + "\n"
	}

	var buff strings.Builder

	out := colorable.NewNonColorable(&buff)

	_, _ = out.Write([]byte(">\n"))

	for line := range strings.Lines(details) {
		_, _ = out.Write([]byte("    "))
		_, _ = out.Write([]byte(line))
	}

	_, _ = out.Write([]byte("\n"))

	return buff.String()
}

func testTextOutput(report *ctrf.Report, output io.Writer) error {
	version := color.New(color.FgBlack).FprintfFunc()
	plan := color.New(color.FgBlack).FprintfFunc()

	failed := color.New(color.FgHiRed).FprintfFunc()
	passed := color.New(color.FgHiGreen).FprintfFunc()

	separator := color.New(color.FgBlack).FprintfFunc()
	description := color.New(color.FgHiWhite).FprintfFunc()

	diag := color.New(color.FgBlack, color.Italic).FprintfFunc()

	version(output, "TAP version 14\n")
	plan(output, "1..%d\n", report.Results.Summary.Tests)

	for idx, test := range report.Results.Tests {
		testIndex := idx + 1

		if test.Status == ctrf.TestPassed {
			passed(output, "ok %d", testIndex)
		} else {
			failed(output, "not ok %d", testIndex)
		}

		separator(output, " - ")
		description(output, "%s\n", test.Filepath)

		if test.Status == ctrf.TestPassed {
			continue
		}

		diag(output, "  ---\n")
		diag(output, "  message: %s", formatDetails(test.Message))
		diag(output, "  ---\n")
	}

	return nil
}

func testJSONOutput(report *ctrf.Report, output io.Writer, compact bool) error {
	encoder := json.NewEncoder(output)

	if !compact {
		encoder.SetIndent("", "  ")
	}

	return encoder.Encode(report)
}
