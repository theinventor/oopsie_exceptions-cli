package cmd

import "github.com/spf13/cobra"

func newProjectCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "project",
		Short: "List, inspect, and pin Oopsie projects",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runProjectGet,
	}
	c.AddCommand(newProjectListCmd())
	c.AddCommand(newProjectGetCmd())
	c.AddCommand(newProjectPinCmd())
	return c
}

func newProjectListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List projects accessible to the current API key",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli := newAPIClient()
			projects, multi, err := fetchProjects(cmd, cli)
			if err != nil {
				return err
			}
			authType := "project"
			if multi {
				authType = "user"
			}
			return printJSON(cmd.OutOrStdout(), map[string]any{
				"auth_type": authType,
				"count":     len(projects),
				"projects":  projects,
			})
		},
	}
}

func newProjectGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [id-or-name]",
		Short: "Show a single project summary",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runProjectGet,
	}
}

func newProjectPinCmd() *cobra.Command {
	var clear bool
	c := &cobra.Command{
		Use:   "pin <id-or-name>|--clear",
		Short: "Pin a default project on the selected saved profile",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if clear && len(args) > 0 {
				return usageError("use either --clear or a project value, not both")
			}
			project := ""
			if len(args) == 1 {
				project = args[0]
			}
			if clear {
				project = ""
			}
			if !clear && project == "" {
				return usageError("missing project id/name or --clear")
			}
			p, err := updateProfileProject(rootProfile, project)
			if err != nil {
				return usageError("%v", err)
			}
			return printJSON(cmd.OutOrStdout(), map[string]any{
				"profile": rootProfile,
				"project": projectObject(p.ProjectID, p.ProjectName),
			})
		},
	}
	c.Flags().BoolVar(&clear, "clear", false, "clear the pinned project")
	return c
}

func runProjectGet(cmd *cobra.Command, args []string) error {
	cli := newAPIClient()
	projects, multi, err := fetchProjects(cmd, cli)
	if err != nil {
		return err
	}
	want := effectiveProjectInput(cli)
	if len(args) == 1 {
		want = args[0]
	}
	if want == "" && len(projects) == 1 {
		return printJSON(cmd.OutOrStdout(), map[string]any{"project": projects[0], "auth_type": authType(multi)})
	}
	if want == "" {
		return usageError("user API key has multiple projects; pass `oopsie project get <id-or-name>` or `--project <id-or-name>`")
	}
	for _, p := range projects {
		if p.ID.String() == want || p.Name == want {
			return printJSON(cmd.OutOrStdout(), map[string]any{"project": p, "auth_type": authType(multi)})
		}
	}
	return usageError("project %q is not accessible", want)
}

func authType(multi bool) string {
	if multi {
		return "user"
	}
	return "project"
}

func newProjectsAliasCmd() *cobra.Command {
	cmd := newProjectListCmd()
	cmd.Use = "projects"
	cmd.Hidden = true
	cmd.Deprecated = "use `oopsie project list`"
	return cmd
}
