package cmd

import "github.com/spf13/cobra"

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current auth and project access",
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
				"authenticated":  true,
				"auth_type":      authType,
				"source":         cli.Source,
				"profile":        cli.ProfileName,
				"storage":        cli.Backend,
				"api_url":        cli.BaseURL,
				"api_key":        cli.MaskedAPIKey(),
				"project":        projectObject(cli.ProjectID, cli.ProjectName),
				"projects_count": len(projects),
				"projects":       projects,
			})
		},
	}
}
