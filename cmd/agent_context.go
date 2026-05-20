package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/theinventor/oopsie_exceptions-cli/internal/client"
	"github.com/theinventor/oopsie_exceptions-cli/internal/config"
	"github.com/theinventor/oopsie_exceptions-cli/internal/enums"
	"github.com/theinventor/oopsie_exceptions-cli/internal/exitcode"
)

const AgentContextSchemaVersion = "1"

func newAgentContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agent-context",
		Short: "Emit a versioned machine-readable description of the CLI",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return printJSON(cmd.OutOrStdout(), buildAgentContext(cmd.Root()))
		},
	}
}

func buildAgentContext(root *cobra.Command) map[string]any {
	commands := map[string]any{}
	for _, c := range root.Commands() {
		if shouldSkipForAgents(c) {
			continue
		}
		commands[c.Name()] = describeCommand(c)
	}
	exitCodes := map[string]string{}
	for _, code := range exitcode.All() {
		exitCodes[intString(code)] = exitcode.Description(code)
	}
	return map[string]any{
		"schema_version":     AgentContextSchemaVersion,
		"cli_version":        Version,
		"commands":           commands,
		"global_flags":       describeFlags(root.PersistentFlags()),
		"enums":              enums.InContext,
		"exit_codes":         exitCodes,
		"available_profiles": availableProfiles(),
		"environment": map[string]string{
			"api_key": client.EnvAPIKey,
			"api_url": client.EnvAPIURL,
			"project": client.EnvProject,
			"config":  "OOPSIE_CONFIG",
		},
		"resources": map[string]any{
			"skill": map[string]any{
				"command":     "oopsie skill get oopsie",
				"description": "Bundled Oopsie agent workflow skill.",
			},
		},
		"api_paths": map[string]string{
			"project":            "/api/v1/project",
			"error_groups":       "/api/v1/error_groups",
			"notification_rules": "/api/v1/notification_rules",
			"exceptions":         "/api/v1/exceptions",
		},
	}
}

func describeCommand(c *cobra.Command) map[string]any {
	desc := map[string]any{"summary": c.Short}
	if c.Long != "" && c.Long != c.Short {
		desc["description"] = c.Long
	}
	if useArgs := strings.TrimSpace(strings.TrimPrefix(c.Use, c.Name())); useArgs != "" {
		desc["args"] = useArgs
	}
	if flags := describeFlags(c.LocalFlags()); len(flags) > 0 {
		desc["flags"] = flags
	}
	subs := map[string]any{}
	for _, sub := range c.Commands() {
		if shouldSkipForAgents(sub) {
			continue
		}
		subs[sub.Name()] = describeCommand(sub)
	}
	if len(subs) > 0 {
		desc["subcommands"] = subs
	}
	return desc
}

func describeFlags(set *pflag.FlagSet) map[string]any {
	flags := map[string]any{}
	set.VisitAll(func(f *pflag.Flag) {
		if f.Hidden || f.Name == "help" {
			return
		}
		entry := map[string]any{
			"type":  f.Value.Type(),
			"usage": f.Usage,
		}
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" && f.DefValue != "[]" {
			entry["default"] = f.DefValue
		}
		if f.Shorthand != "" {
			entry["shorthand"] = f.Shorthand
		}
		flags["--"+f.Name] = entry
	})
	return flags
}

func shouldSkipForAgents(c *cobra.Command) bool {
	return c.Hidden || c.Name() == "help" || c.Name() == "completion"
}

func availableProfiles() []string {
	f, err := config.Load()
	if err != nil || f.Profiles == nil {
		return []string{}
	}
	return f.Names()
}
