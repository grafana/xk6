package cmd

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"github.com/spf13/cobra"
	"github.com/szkiba/efa"
	"go.k6.io/xk6/internal/lint"
)

var (
	//go:embed help/lint.md
	lintHelp string

	//go:embed help/lint-example.txt
	lintExample string

	//go:embed help/checks.md
	checksHelp string

	//go:embed help/presets.md
	presetsHelp string

	errLintingFailed = errors.New("linting failed")
)

func validPresetIDs() []string {
	ids := make([]string, len(lint.PresetIDs))

	for i, id := range lint.PresetIDs {
		ids[i] = string(id)
	}

	return ids
}

var errInvalidPresetID = errors.New("valid values are: " + strings.Join(validPresetIDs(), ", "))

type presetID lint.PresetID

func (p *presetID) String() string {
	return string(*p)
}

func (p *presetID) Set(v string) error {
	id, err := lint.ParsePresetID(v)
	if err != nil {
		return errInvalidPresetID
	}

	*p = presetID(id)

	return nil
}

func (p *presetID) Type() string {
	return "preset"
}

func validCheckIDs() []string {
	ids := make([]string, len(lint.CheckIDs))

	for i, id := range lint.CheckIDs {
		ids[i] = string(id)
	}

	return ids
}

var errInvalidCheckID = errors.New("valid values are: " + strings.Join(validCheckIDs(), ", "))

type checkIDs []lint.CheckID

func (c *checkIDs) String() string {
	strs := make([]string, len(*c))

	for i, id := range *c {
		strs[i] = string(id)
	}

	return strings.Join(strs, ", ")
}

func (c *checkIDs) Set(v string) error {
	names := strings.Split(v, ",")

	ids := make([]lint.CheckID, 0, len(names))

	for _, name := range names {
		id, err := lint.ParseCheckID(strings.TrimSpace(name))
		if err != nil {
			return errInvalidCheckID
		}

		ids = append(ids, id)
	}

	*c = ids

	return nil
}

func (c *checkIDs) Type() string {
	return "checkers"
}

type options struct {
	out        string
	compact    bool
	quiet      bool
	json       bool
	preset     presetID
	enable     checkIDs
	disable    checkIDs
	enableOnly checkIDs
	k6version  string
	k6repo     string
}

func lintCmd() *cobra.Command {
	opts := new(options)

	opts.preset = presetID(lint.PresetIDLoose)

	cmd := &cobra.Command{
		Use:           "lint [flags] [directory]",
		Short:         shortHelp(lintHelp),
		Long:          lintHelp,
		Example:       lintExample,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			opts.json = opts.json || opts.compact
			opts.quiet = cmd.Flags().Lookup("quiet").Changed

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

	flags.StringVarP(&opts.out, "out", "o", "", "Write output to file instead of stdout")
	flags.BoolVar(&opts.json, "json", false, "Generate JSON output")
	flags.BoolVarP(&opts.compact, "compact", "c", false, "Compact instead of pretty-printed JSON output")
	flags.VarP(&opts.preset, "preset", "p", "Check preset to use (default: loose)")
	flags.Var(&opts.enable, "enable", "Enable additional checks (comma-separated list)")
	flags.Var(&opts.disable, "disable", "Disable specific checks (comma-separated list)")
	flags.Var(&opts.enableOnly, "enable-only", "Enable only specified checks, ignoring preset (comma-separated list)")

	flags.StringVarP(&opts.k6version, "k6-version", "k", defaultK6Version, "The k6 version to use for build")
	flags.StringVar(&opts.k6repo, "k6-repo", defaultK6Repo, "The k6 repository to use for the build")

	env := efa.New(flags, appname+"_"+cmd.Name(), nil)

	cobra.CheckErr(env.Bind("preset", "enable", "disable", "enable-only"))

	cmd.AddCommand(helpTopic("checks", checksHelp))
	cmd.AddCommand(helpTopic("presets", presetsHelp))

	return cmd
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

	lopts := lint.Options{
		Preset:     lint.PresetID(opts.preset),
		Enable:     opts.enable,
		Disable:    opts.disable,
		EnableOnly: opts.enableOnly,
	}

	compliance, err := lint.Lint(ctx, dir, &lopts)
	if err != nil {
		return err
	}

	return lintOutput(opts.quiet, opts.json, compliance, output, opts.compact)
}

func lintOutput(quiet bool, json bool, compliance *lint.Compliance, output io.Writer, compact bool) error {
	if json {
		logOutput(compliance)

		return jsonOutput(compliance, output, compact)
	}

	if quiet {
		logOutput(compliance)
	} else {
		textOutput(compliance, output)
	}

	if !compliance.Passed {
		return errLintingFailed
	}

	return nil
}

func logOutput(compliance *lint.Compliance) {
	for _, check := range compliance.Checks {
		if check.Passed {
			slog.Info("Check passed", "check", check.ID, "details", check.Details)

			continue
		}

		slog.Warn("Check failed", "check", check.ID, "details", check.Details, "resolution", check.Definition.Resolution)
	}
}

func jsonOutput(compliance any, output io.Writer, compact bool) error {
	encoder := json.NewEncoder(output)

	if !compact {
		encoder.SetIndent("", "  ")
	}

	return encoder.Encode(compliance)
}

func textOutput(compliance *lint.Compliance, output io.Writer) {
	heading := color.New(color.FgHiWhite, color.Bold).FprintfFunc()

	failed := color.New(color.FgRed).FprintfFunc()
	passed := color.New(color.FgGreen).FprintfFunc()
	plain := color.New(color.FgWhite).FprintfFunc()
	rationale := color.New(color.Italic).FprintfFunc()
	resolution := color.New(color.Bold).FprintfFunc()

	heading(output, "k6 extension compliance\n\n")

	for _, check := range compliance.Checks {
		fprintf := failed
		details := failed
		symbol := "✗"

		if check.Passed {
			fprintf = passed
			details = plain
			symbol = "✔"
		}

		fprintf(output, "%s %-20s\n", symbol, check.ID)

		if len(check.Details) != 0 {
			details(output, "  %s\n", check.Details)
		}

		if !check.Passed && check.Definition != nil {
			if len(check.Definition.Rationale) != 0 {
				rationale(output, "\n  %s\n", check.Definition.Rationale)
			}

			if len(check.Definition.Resolution) != 0 {
				resolution(output, "\n %s\n", check.Definition.Resolution)
			}
		}
	}

	plain(output, "\n")
}

var errNotDirectory = errors.New("not a directory")
