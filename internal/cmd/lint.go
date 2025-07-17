package cmd

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"github.com/spf13/cobra"
	"go.k6.io/xk6/internal/lint"
)

var (
	//go:embed help/lint.md
	lintHelp string

	//go:embed help/checkers.md
	checkersHelp string
)

type options struct {
	out       string
	compact   bool
	quiet     bool
	json      bool
	passing   lint.Grade
	passedStr []string
	passed    []lint.Checker
	official  bool
}

func lintCmd() *cobra.Command {
	opts := new(options)
	opts.passing = lint.GradeC

	cmd := &cobra.Command{
		Use:           "lint [flags] [directory]",
		Short:         shortHelp(lintHelp),
		Long:          lintHelp,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			opts.json = opts.json || opts.compact
			opts.quiet = cmd.Flags().Lookup("quiet").Changed

			var err error

			opts.passed, err = parseCheckers(opts.passedStr)
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return lintRunE(cmd.Context(), args, opts)
		},
		DisableAutoGenTag: true,
	}

	cmd.SetContext(context.TODO())

	flags := cmd.Flags()

	flags.SortFlags = false

	flags.Var(&opts.passing, "passing", "Set lowest passing grade")
	flags.BoolVar(&opts.official, "official", false, "Enable extra checks for official extensions")
	flags.StringVarP(&opts.out, "out", "o", "", "Write output to file instead of stdout")
	flags.BoolVar(&opts.json, "json", false, "Generate JSON output")
	flags.BoolVarP(&opts.compact, "compact", "c", false, "Compact instead of pretty-printed JSON output")
	flags.StringSliceVar(&opts.passedStr, "passed", []string{}, "Set checker(s) to passed")

	//nolint:errcheck
	flags.MarkHidden("passed") // #nosec G104

	cmd.AddCommand(helpTopic("checkers", checkersHelp))

	return cmd
}

func parseCheckers(names []string) ([]lint.Checker, error) {
	checkers := make([]lint.Checker, 0, len(names))

	for _, name := range names {
		checker, err := lint.ParseChecker(name)
		if err != nil {
			return nil, err
		}

		checkers = append(checkers, checker)
	}

	return checkers, nil
}

func dirArg(args []string) (string, error) {
	var dir string

	if len(args) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}

		dir = cwd
	} else {
		dir = args[0]
	}

	info, err := os.Stat(dir)
	if err != nil {
		return "", err
	}

	if !info.IsDir() {
		return "", fmt.Errorf("%w: %s", errNotDirectory, dir)
	}

	return dir, nil
}

func lintRunE(ctx context.Context, args []string, opts *options) (result error) {
	dir, err := dirArg(args)
	if err != nil {
		return err
	}

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

		output = file
	}

	compliance, err := lint.Lint(ctx, dir, &lint.Options{Passed: opts.passed, Official: opts.official})
	if err != nil {
		return err
	}

	if opts.quiet {
		return nil
	}

	if opts.json {
		return jsonOutput(compliance, output, opts.compact)
	}

	textOutput(compliance, output)

	if string(compliance.Grade) > string(opts.passing) {
		return fmt.Errorf("%w: %s", errFailingGrade, compliance.Grade)
	}

	return nil
}

func jsonOutput(compliance any, output io.Writer, compact bool) error {
	encoder := json.NewEncoder(output)

	if !compact {
		encoder.SetIndent("", "  ")
	}

	return encoder.Encode(compliance)
}

func gradeColor(grade lint.Grade) *color.Color {
	var c color.Attribute

	switch grade {
	case lint.GradeA:
		c = color.BgHiGreen
	case lint.GradeB:
		c = color.BgGreen
	case lint.GradeC:
		c = color.BgHiCyan
	case lint.GradeD:
		c = color.BgYellow
	case lint.GradeE:
		c = color.BgMagenta
	case lint.GradeF, lint.GradeZ:
		c = color.BgRed
	case lint.GradeG:
		fallthrough
	default:
		c = color.BgBlue
	}

	return color.New(c)
}

func textOutput(compliance *lint.Compliance, output io.Writer) {
	heading := color.New(color.FgHiWhite, color.Bold).FprintfFunc()

	value := gradeColor(compliance.Grade).FprintfFunc()
	label := color.New(color.FgBlack, color.BgWhite).FprintfFunc()
	details := color.New(color.Italic).FprintfFunc()
	failed := color.New(color.FgRed).FprintfFunc()
	passed := color.New(color.FgGreen).FprintfFunc()
	plain := color.New(color.FgWhite).FprintfFunc()

	heading(output, "k6 extension compliance\n──────────┬─────────────\n")
	label(output, " grade ")
	value(output, " %s ", string(compliance.Grade))
	heading(output, "│")
	label(output, " level ")
	value(output, " %3d%% ", compliance.Level)
	plain(output, "\n\n")

	heading(output, "Details\n───────\n")

	for _, check := range compliance.Checks {
		fprintf := failed
		symbol := "✗"

		if check.Passed {
			fprintf = passed
			symbol = "✔"
		}

		fprintf(output, "%s %-20s\n", symbol, check.ID)

		if len(check.Details) != 0 {
			details(output, "  %s\n", check.Details)
		}
	}

	plain(output, "\n")
}

var (
	errNotDirectory = errors.New("not a directory")
	errFailingGrade = errors.New("failing grade")
)
