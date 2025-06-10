package cmd

import (
	_ "embed"
	"strings"

	"github.com/spf13/cobra"
)

var (
	//go:embed help/features.md
	featuresHelp string

	//go:embed help/install.md
	installHelp string

	//go:embed help/devcontainers.md
	devcontainersHelp string

	//go:embed help/docker.md
	dockerHelp string
)

func helpTopics() []*cobra.Command {
	return []*cobra.Command{
		helpTopic("features", featuresHelp),
		helpTopic("devcontainers", devcontainersHelp),
		helpTopic("docker", dockerHelp),
		helpTopic("install", installHelp),
	}
}

func helpTopic(name, long string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: shortHelp(long),
		Long:  long,
	}
}

func shortHelp(long string) string {
	short, _, _ := strings.Cut(long, "\n")

	return strings.TrimSpace(short)
}
