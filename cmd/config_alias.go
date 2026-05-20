package cmd

import (
	"github.com/spf13/cobra"
	"github.com/theinventor/oopsie_exceptions-cli/internal/client"
	"github.com/theinventor/oopsie_exceptions-cli/internal/config"
	"github.com/theinventor/oopsie_exceptions-cli/internal/credstore"
)

func newConfigAliasCmd() *cobra.Command {
	c := &cobra.Command{
		Use:        "config",
		Short:      "Deprecated compatibility aliases for the old bash CLI",
		Hidden:     true,
		Deprecated: "use `oopsie auth ...` and `oopsie project pin ...`",
	}
	c.AddCommand(newConfigAddAliasCmd())
	c.AddCommand(newConfigListAliasCmd())
	c.AddCommand(newConfigUseAliasCmd())
	c.AddCommand(newConfigRemoveAliasCmd())
	c.AddCommand(newConfigSetProjectAliasCmd())
	return c
}

func newConfigAddAliasCmd() *cobra.Command {
	var server, key, project string
	c := &cobra.Command{
		Use:   "add <name>",
		Short: "Deprecated alias for auth save",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || key == "" {
				return usageError("usage: oopsie config add <name> --server <url> --key <api_key> [--project <id-or-name>]")
			}
			backend, err := resolveStorage("")
			if err != nil {
				return err
			}
			p := config.Profile{APIURL: server}
			if project != "" {
				if isNumericID(project) {
					p.ProjectID = project
				} else {
					p.ProjectName = project
				}
			}
			canon, err := persistProfile(args[0], p, key, backend)
			if err != nil {
				return err
			}
			return printJSON(cmd.OutOrStdout(), map[string]any{
				"saved":   true,
				"profile": args[0],
				"api_url": server,
				"storage": credstore.Describe(canon),
				"api_key": client.MaskKey(key),
				"project": projectObject(p.ProjectID, p.ProjectName),
			})
		},
	}
	c.Flags().StringVar(&server, "server", "", "Oopsie server URL")
	c.Flags().StringVar(&key, "key", "", "Oopsie API key")
	c.Flags().StringVar(&project, "project", "", "default project id or exact name")
	c.Flags().StringVar(&project, "pin", "", "alias for --project")
	return c
}

func newConfigListAliasCmd() *cobra.Command {
	cmd := newAuthListCmd()
	cmd.Use = "list"
	return cmd
}

func newConfigUseAliasCmd() *cobra.Command {
	cmd := newAuthUseCmd()
	cmd.Use = "use <profile>"
	return cmd
}

func newConfigRemoveAliasCmd() *cobra.Command {
	cmd := newAuthLogoutCmd()
	cmd.Use = "remove <profile>"
	return cmd
}

func newConfigSetProjectAliasCmd() *cobra.Command {
	var clear bool
	c := &cobra.Command{
		Use:   "set-project <id-or-name>|--clear",
		Short: "Deprecated alias for project pin",
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
			return printJSON(cmd.OutOrStdout(), map[string]any{"project": projectObject(p.ProjectID, p.ProjectName)})
		},
	}
	c.Flags().BoolVar(&clear, "clear", false, "clear the pinned project")
	return c
}
