package docsme

import (
	"strings"
	"sync"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var templateFuncsOnce sync.Once //nolint:gochecknoglobals

// SetUsageTemplate adds an "Environment" section to the usage template of the command.
// The Environment section contains a list of environment variable names and descriptions.
// The list is taken from the flags of the command. If the flag has an annotation named "environment",
// its value will be the name of the environment variable and the flag usage will be the description.
func SetUsageTemplate(cmd *cobra.Command) {
	templateFuncsOnce.Do(func() {
		cobra.AddTemplateFuncs(templateFuncs())
	})

	tmpl := strings.Replace(cmd.UsageTemplate(), "{{if .HasHelpSubCommands}}", `{{if hasEnvironment .}}

Environment:{{range $envvar := (envVariables .)}}
  {{rpad $envvar.Name (envPadding $) }}{{$envvar.Usage}}{{end}}{{end}}{{if .HasHelpSubCommands}}`, 1)

	cmd.SetUsageTemplate(tmpl)
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"hasEnvironment": hasEnvironment,
		"envPadding":     envPadding,
		"envVariables":   envVariables,
	}
}

func hasEnvironment(cmd *cobra.Command) bool {
	found := false

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		found = found || len(envVariableFromAnnotation(f)) > 0
	})

	return found
}

func envPadding(cmd *cobra.Command) int {
	padding := 0

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if name := envVariableFromAnnotation(f); len(name) > padding {
			padding = len(name)
		}
	})

	const minPadding = 3

	return padding + minPadding
}

type variable struct {
	Name  string
	Usage string
}

func envVariables(cmd *cobra.Command) []*variable {
	all := make([]*variable, 0)

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if name := envVariableFromAnnotation(f); len(name) != 0 {
			all = append(all, &variable{Name: name, Usage: f.Usage})
		}
	})

	return all
}

func envVariableFromAnnotation(flag *pflag.Flag) string {
	if len(flag.Annotations) == 0 {
		return ""
	}

	if value, has := flag.Annotations["environment"]; has && len(value) != 0 {
		return value[0]
	}

	return ""
}
