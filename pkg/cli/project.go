package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/project"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewProject(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Project{client: c.ClientFactory}, cobra.Command{
		Use:     "project [flags]",
		Aliases: []string{"projects"},
		Example: `
acorn project`,
		SilenceUsage:      true,
		Short:             "Manage projects",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, projectsCompletion).complete,
	})
	cmd.AddCommand(NewProjectCreate(c))
	cmd.AddCommand(NewProjectRm(c))
	cmd.AddCommand(NewProjectUse(c))
	return cmd
}

type Project struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client ClientFactory
}

type projectEntry struct {
	Name        string `json:"name,omitempty"`
	Default     bool   `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

func (a *Project) Run(cmd *cobra.Command, args []string) error {
	cfg, err := config.ReadCLIConfig()
	if err != nil {
		return err
	}

	var projectNames []string
	if len(args) == 1 {
		_, err := project.Get(cmd.Context(), a.client.Options().WithCLIConfig(cfg), args[0])
		if err != nil {
			return err
		}
		projectNames = append(projectNames, args[0])
	} else {
		projects, err := project.List(cmd.Context(), a.client.Options().WithCLIConfig(cfg))
		if err != nil {
			return err
		}
		if len(args) == 0 {
			projectNames = append(projectNames, projects...)
		} else {
			for _, arg := range args {
				if slices.Contains(projects, arg) {
					projectNames = append(projectNames, arg)
				}
			}
		}
	}

	defaultProject := cfg.CurrentProject

	c, err := project.Client(cmd.Context(), a.client.Options())
	if err == nil {
		defaultProject = c.GetProject()
	}

	out := table.NewWriter(tables.ProjectClient, a.Quiet, a.Output)
	for _, project := range projectNames {
		out.Write(projectEntry{
			Name:    project,
			Default: defaultProject == project,
		})
	}

	for _, entry := range typed.Sorted(cfg.ProjectAliases) {
		out.Write(projectEntry{
			Name:        entry.Key,
			Default:     defaultProject == entry.Value,
			Description: fmt.Sprintf("alias to %s", entry.Value),
		})
	}

	return out.Err()
}
