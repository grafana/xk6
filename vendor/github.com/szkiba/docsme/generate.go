package docsme

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type mdgen struct {
	offset int
}

const (
	h1 = 1
	h2 = 2
	h3 = 3
)

func (m *mdgen) generate(cmd *cobra.Command, w io.Writer) error {
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	buff := new(bytes.Buffer)

	m.name(cmd, buff)
	m.short(cmd, buff)
	m.long(cmd, buff)
	m.useLine(cmd, buff)
	m.examples(cmd, buff)
	m.additionalHelpTopics(cmd, buff)
	m.flags(cmd, buff)
	m.environment(cmd, buff)
	m.seeAlso(cmd, buff)

	if err := m.subcommands(cmd, buff); err != nil {
		return err
	}

	_, err := buff.WriteTo(w)

	return err
}

func (m *mdgen) subcommands(cmd *cobra.Command, buff *bytes.Buffer) error {
	for _, child := range cmd.Commands() {
		if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
			continue
		}

		buff.WriteString("---\n\n")

		if err := m.generate(child, buff); err != nil {
			return err
		}
	}

	return nil
}

func (m *mdgen) name(cmd *cobra.Command, buff *bytes.Buffer) {
	buff.WriteString(m.heading(h1, cmd.CommandPath()))
}

func (m *mdgen) short(cmd *cobra.Command, buff *bytes.Buffer) {
	if cmd.Runnable() {
		buff.WriteString(cmd.Short + "\n\n")
	} else {
		buff.WriteString("**" + cmd.Short + "**\n\n")
	}
}

func (m *mdgen) long(cmd *cobra.Command, buff *bytes.Buffer) {
	if len(cmd.Long) == 0 {
		return
	}

	if cmd.Runnable() {
		buff.WriteString(m.heading(h2, "Synopsis"))
	}

	buff.WriteString(getLong(cmd) + "\n\n")
}

func (m *mdgen) useLine(cmd *cobra.Command, buff *bytes.Buffer) {
	if cmd.Runnable() {
		fmt.Fprint(buff, m.heading(h2, "Usage"))
		fmt.Fprintf(buff, "```bash\n%s\n```\n\n", cmd.UseLine())
	}
}

func (m *mdgen) flags(cmd *cobra.Command, buff *bytes.Buffer) {
	if !cmd.Runnable() {
		return
	}

	formatFlags := func(flags *pflag.FlagSet, title string) {
		flags.SetOutput(buff)

		if flags.HasAvailableFlags() {
			buff.WriteString(m.heading(h2, title))
			buff.WriteString("```\n")
			flags.PrintDefaults()
			buff.WriteString("```\n\n")
		}
	}

	formatFlags(cmd.NonInheritedFlags(), "Flags")
	formatFlags(cmd.InheritedFlags(), "Global Flags")
}

func (m *mdgen) environment(cmd *cobra.Command, buff *bytes.Buffer) {
	if !hasEnvironment(cmd) {
		return
	}

	buff.WriteString(m.heading(h2, "Environment"))
	fmt.Fprintln(buff, "```")

	padding := envPadding(cmd)

	for _, envvar := range envVariables(cmd) {
		fmt.Fprintf(buff, "  %-*s   %s\n", padding, envvar.Name, envvar.Usage)
	}

	fmt.Fprint(buff, "```\n\n")
}

func (m *mdgen) examples(cmd *cobra.Command, buff *bytes.Buffer) {
	if len(cmd.Example) == 0 {
		return
	}

	buff.WriteString(m.heading(h2, "Examples"))
	fmt.Fprintf(buff, "```\n%s\n```\n\n", cmd.Example)
}

func (m *mdgen) additionalHelpTopics(cmd *cobra.Command, buff *bytes.Buffer) {
	for _, child := range cmd.Commands() {
		if !child.IsAdditionalHelpTopicCommand() {
			continue
		}

		head := strings.TrimSpace(child.Short)
		if len(head) == 0 {
			head = child.Use
		}

		buff.WriteString(m.heading(h3, head))

		body := child.Long
		if len(body) > 0 {
			if strings.HasPrefix(body, head+"\n") {
				body = strings.TrimSpace(strings.TrimPrefix(body, head))
			}

			buff.WriteString(body + "\n\n")
		}
	}
}

func (m *mdgen) seeAlso(cmd *cobra.Command, buff *bytes.Buffer) {
	if !hasSeeAlso(cmd) {
		return
	}

	if cmd.HasParent() {
		buff.WriteString(m.heading(h2, "SEE ALSO"))

		parent := cmd.Parent()
		pname := parent.CommandPath()
		link := strings.ReplaceAll(pname, " ", "-")
		fmt.Fprintf(buff, "* [%s](#%s)\t - %s\n", pname, link, parent.Short)
	}

	firstChild := true

	for _, child := range cmd.Commands() {
		if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
			continue
		}

		if firstChild {
			buff.WriteString(m.heading(h2, "Commands"))

			firstChild = false
		}

		cname := cmd.CommandPath() + " " + child.Name()
		link := strings.ReplaceAll(cname, " ", "-")
		fmt.Fprintf(buff, "* [%s](#%s)\t - %s\n", cname, link, child.Short)
	}

	buff.WriteString("\n")
}

func (m *mdgen) heading(level int, value string) string {
	return strings.Repeat("#", m.offset+level) + " " + value + "\n\n"
}

func hasSeeAlso(cmd *cobra.Command) bool {
	if cmd.HasParent() {
		return true
	}

	for _, c := range cmd.Commands() {
		if c.IsAvailableCommand() && !c.IsAdditionalHelpTopicCommand() {
			return true
		}
	}

	return false
}

func getLong(cmd *cobra.Command) string {
	if len(cmd.Short) == 0 {
		return cmd.Long
	}

	if !strings.HasPrefix(cmd.Long, cmd.Short+"\n") {
		return cmd.Long
	}

	return strings.TrimSpace(strings.TrimPrefix(cmd.Long, cmd.Short))
}
