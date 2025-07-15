// Package cmd contains the Cobra Command that implements the xk6 CLI.
package cmd

import (
	"context"
	_ "embed"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/szkiba/docsme"
)

// Execute executes root command.
func Execute() {
	cobra.EnableCommandSorting = false

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go trapSignals(ctx, cancel)

	cmd := New(initLogging())

	cmd.SetContext(ctx)

	err := cmd.Execute()
	if err != nil {
		slog.Error(err.Error())
		cancel()
		os.Exit(1) //nolint:gocritic
	}
}

func trapSignals(ctx context.Context, cancel context.CancelFunc) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	select {
	case <-sig:
		slog.Info("SIGINT: Shutting down")
		cancel()
	case <-ctx.Done():
		return
	}
}

//nolint:gochecknoglobals
var (
	version = ""
	appname = "xk6"
	// Note that binary = appname will not work because ldflags inserts the values to the same place.
	binary = "xk6"
)

//go:embed help/root.md
var rootHelp string

// New creates a new instance of Cobra Command, which implements the xk6 CLI.
func New(levelVar *slog.LevelVar) *cobra.Command {
	root := &cobra.Command{
		Use:               appname,
		Version:           getVersion(),
		Short:             shortHelp(rootHelp),
		Long:              rootHelp,
		Args:              cobra.NoArgs,
		SilenceUsage:      true,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	if binary != appname {
		root.Annotations = map[string]string{cobra.CommandDisplayNameAnnotation: strings.ReplaceAll(binary, "-", " ")}
	}

	flags := root.Flags()

	flags.BoolP("version", "V", false, "Version for "+appname)

	gflags := root.PersistentFlags()

	gflags.BoolP("help", "h", false, "Help about any command ")

	quiet := gflags.BoolP("quiet", "q", false, "Suppress output")
	verbose := gflags.BoolP("verbose", "v", false, "Verbose output")

	root.MarkFlagsMutuallyExclusive("quiet", "verbose")

	root.AddCommand(versionCmd(), newCmd(), buildCmd(), runCmd(), lintCmd(), syncCmd())
	root.AddCommand(helpTopics()...)

	cmd := adjustCmd()
	cmd.Hidden = true // This is an internal command, so we don't want to show it in the help output.
	root.AddCommand(cmd)

	if levelVar != nil {
		root.PersistentPreRun = func(_ *cobra.Command, _ []string) {
			levelVar.Set(toLogLevel(*quiet, *verbose))
		}
	}

	docsme.SetUsageTemplate(root)

	return root
}

func getVersion() string {
	if len(version) != 0 {
		return version
	}

	var (
		commit string
		dirty  string
		suffix string
	)

	buildInfo, ok := debug.ReadBuildInfo()
	if ok {
		for _, s := range buildInfo.Settings {
			switch s.Key {
			case "vcs.revision":
				const shortCommitLen = 7
				commit = s.Value[:min(len(s.Value), shortCommitLen)]
			case "vcs.modified":
				if s.Value == "true" {
					dirty = ".dirty"
				}
			default:
			}
		}
	}

	if len(commit) != 0 {
		suffix = "+" + commit + dirty
	}

	return "0.0.1-next" + suffix
}

func initLogging() *slog.LevelVar {
	levelVar := new(slog.LevelVar)

	w := os.Stderr

	term := isatty.IsTerminal(w.Fd())

	topts := &tint.Options{
		NoColor:    !term || os.Getenv("NO_COLOR") == "true",
		TimeFormat: time.RFC3339,
		Level:      levelVar,
	}

	if term {
		topts.TimeFormat = time.Kitchen
	}

	logger := slog.New(tint.NewHandler(colorable.NewColorable(w), topts))

	slog.SetDefault(logger)

	return levelVar
}

func toLogLevel(quiet, verbose bool) slog.Level {
	switch {
	case quiet:
		return slog.LevelWarn
	case verbose:
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}
