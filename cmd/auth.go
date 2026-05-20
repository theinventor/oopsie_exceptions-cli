package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/theinventor/oopsie_exceptions-cli/internal/client"
	"github.com/theinventor/oopsie_exceptions-cli/internal/config"
	"github.com/theinventor/oopsie_exceptions-cli/internal/credstore"
)

const storageFlagDescription = "where to persist the API key: auto (default; keychain if available, else file), keychain, or file"

func newAuthCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "auth",
		Short: "Manage saved credentials and profiles",
		Long: `Manage Oopsie profiles. Environment variables OOPSIE_API_KEY and
OOPSIE_API_URL win over the saved default profile unless --profile is passed.`,
	}
	c.AddCommand(newAuthSaveCmd())
	c.AddCommand(newAuthStatusCmd())
	c.AddCommand(newAuthListCmd())
	c.AddCommand(newAuthUseCmd())
	c.AddCommand(newAuthLogoutCmd())
	c.AddCommand(newAuthMigrateCmd())
	return c
}

func resolveStorage(storage string) (string, error) {
	backend, err := credstore.ResolveBackend(storage)
	if err != nil {
		return "", usageError("%v", err)
	}
	return backend, nil
}

func persistProfile(name string, p config.Profile, secret, backend string) (string, error) {
	canon, err := credstore.Put(name, backend, secret)
	if err != nil {
		return "", err
	}
	switch canon {
	case credstore.BackendKeychain:
		p.APIKey = ""
	case credstore.BackendFile:
		p.APIKey = secret
	}
	p.Backend = canon

	f, err := config.Load()
	if err != nil {
		return "", err
	}
	f.Put(name, p)
	if err := f.Save(); err != nil {
		return "", err
	}
	return canon, nil
}

func newAuthSaveCmd() *cobra.Command {
	var profileName, apiKey, apiURL, project, storage string
	c := &cobra.Command{
		Use:   "save",
		Short: "Save an existing Oopsie API key as a profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if profileName == "" || apiKey == "" {
				return usageError("--profile and --api-key are both required")
			}
			if apiURL == "" {
				apiURL = rootAPIURL
			}
			if apiURL == "" {
				apiURL = os.Getenv(client.EnvAPIURL)
			}
			if apiURL == "" {
				return usageError("--api-url is required for self-hosted Oopsie instances")
			}
			backend, err := resolveStorage(storage)
			if err != nil {
				return err
			}

			p := config.Profile{APIURL: apiURL}
			if project != "" {
				if isNumericID(project) {
					p.ProjectID = project
				} else {
					p.ProjectName = project
				}
			}
			canon, err := persistProfile(profileName, p, apiKey, backend)
			if err != nil {
				return err
			}
			f, _ := config.Load()
			return printJSON(cmd.OutOrStdout(), map[string]any{
				"saved":           true,
				"profile":         profileName,
				"default_profile": f.DefaultProfile,
				"storage":         credstore.Describe(canon),
				"api_url":         apiURL,
				"project":         projectObject(p.ProjectID, p.ProjectName),
				"api_key":         client.MaskKey(apiKey),
				"config":          config.Path(),
			})
		},
	}
	c.Flags().StringVar(&profileName, "profile", "", "profile name to save under")
	c.Flags().StringVar(&apiKey, "api-key", "", "Oopsie user or project API key")
	c.Flags().StringVar(&apiURL, "api-url", "", "Oopsie API base URL, e.g. https://oopsie.example.com")
	c.Flags().StringVar(&project, "project", "", "optional default project id or exact project name for this profile")
	c.Flags().StringVar(&storage, "storage", "", storageFlagDescription)
	return c
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the active auth profile without printing secrets",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli := newAPIClient()
			out := map[string]any{
				"authenticated": cli.APIKey != "",
				"source":        cli.Source,
				"profile":       cli.ProfileName,
				"storage":       credstore.Describe(cli.Backend),
				"api_url":       cli.BaseURL,
				"api_key":       cli.MaskedAPIKey(),
				"project":       projectObject(cli.ProjectID, cli.ProjectName),
				"config":        config.Path(),
			}
			if cli.APIKey == "" || cli.BaseURL == "" {
				out["hint"] = "run `oopsie auth save --profile <name> --api-key <key> --api-url <url>` or set OOPSIE_API_KEY and OOPSIE_API_URL"
			}
			return printJSON(cmd.OutOrStdout(), out)
		},
	}
}

func newAuthListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved profiles without printing raw API keys",
		RunE: func(cmd *cobra.Command, _ []string) error {
			f, err := config.Load()
			if err != nil {
				return err
			}
			profiles := make([]map[string]any, 0, len(f.Profiles))
			for _, name := range f.Names() {
				p := f.Profiles[name]
				fp := ""
				if secret, err := credstore.Get(name, p.Backend, p.APIKey); err == nil {
					fp = client.MaskKey(secret)
				}
				profiles = append(profiles, map[string]any{
					"name":       name,
					"default":    name == f.DefaultProfile,
					"api_url":    p.APIURL,
					"storage":    credstore.Describe(p.Backend),
					"api_key":    fp,
					"project":    projectObject(p.ProjectID, p.ProjectName),
					"created_at": p.CreatedAt,
				})
			}
			return printJSON(cmd.OutOrStdout(), map[string]any{
				"default_profile": f.DefaultProfile,
				"profiles":        profiles,
				"config":          config.Path(),
			})
		},
	}
}

func newAuthUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <profile>",
		Short: "Set the default profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := config.Load()
			if err != nil {
				return err
			}
			if err := f.SetDefault(args[0]); err != nil {
				return usageError("%v", err)
			}
			if err := f.Save(); err != nil {
				return err
			}
			return printJSON(cmd.OutOrStdout(), map[string]any{"default_profile": args[0], "config": config.Path()})
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout [profile]",
		Short: "Remove a saved profile",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := config.Load()
			if err != nil {
				return err
			}
			name := rootProfile
			if len(args) == 1 {
				name = args[0]
			}
			if name == "" {
				name = f.DefaultProfile
			}
			if name == "" {
				return usageError("no profile selected")
			}
			p, ok := f.Profiles[name]
			if !ok {
				return usageError("no profile named %q", name)
			}
			if err := credstore.Delete(name, p.Backend); err != nil {
				return err
			}
			_ = f.Delete(name)
			if err := f.Save(); err != nil {
				return err
			}
			return printJSON(cmd.OutOrStdout(), map[string]any{"removed": true, "profile": name, "default_profile": f.DefaultProfile})
		},
	}
}

func newAuthMigrateCmd() *cobra.Command {
	var profileName string
	var all bool
	c := &cobra.Command{
		Use:   "migrate (--profile <name> | --all)",
		Short: "Move file-backed profile secrets into the OS keychain",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if all == (profileName != "") {
				return usageError("use exactly one of --profile or --all")
			}
			backend, err := resolveStorage(credstore.BackendKeychain)
			if err != nil {
				return err
			}
			f, err := config.Load()
			if err != nil {
				return err
			}
			names := []string{profileName}
			if all {
				names = f.Names()
			}
			sort.Strings(names)
			results := []map[string]any{}
			for _, name := range names {
				p, ok := f.Profiles[name]
				if !ok {
					results = append(results, map[string]any{"profile": name, "status": "missing"})
					continue
				}
				if p.Backend == credstore.BackendKeychain {
					results = append(results, map[string]any{"profile": name, "status": "skipped", "reason": "already keychain"})
					continue
				}
				secret, err := credstore.Get(name, p.Backend, p.APIKey)
				if err != nil || secret == "" {
					results = append(results, map[string]any{"profile": name, "status": "skipped", "reason": "no file-backed secret"})
					continue
				}
				canon, err := persistProfile(name, p, secret, backend)
				if err != nil {
					return err
				}
				results = append(results, map[string]any{"profile": name, "status": "migrated", "storage": credstore.Describe(canon)})
			}
			return printJSON(cmd.OutOrStdout(), map[string]any{"profiles": results, "config": config.Path()})
		},
	}
	c.Flags().StringVar(&profileName, "profile", "", "profile to migrate")
	c.Flags().BoolVar(&all, "all", false, "migrate every file-backed profile")
	return c
}

func projectObject(id, name string) map[string]string {
	out := map[string]string{}
	if id != "" {
		out["id"] = id
	}
	if name != "" {
		out["name"] = name
	}
	return out
}

func updateProfileProject(profileName, project string) (config.Profile, error) {
	f, err := config.Load()
	if err != nil {
		return config.Profile{}, err
	}
	if profileName == "" {
		profileName = f.DefaultProfile
	}
	if profileName == "" {
		return config.Profile{}, fmt.Errorf("no profile selected")
	}
	p, ok := f.Profiles[profileName]
	if !ok {
		return config.Profile{}, fmt.Errorf("no profile named %q", profileName)
	}
	p.ProjectID = ""
	p.ProjectName = ""
	if project != "" {
		if isNumericID(project) {
			p.ProjectID = project
		} else {
			p.ProjectName = project
		}
	}
	f.Profiles[profileName] = p
	if err := f.Save(); err != nil {
		return config.Profile{}, err
	}
	return p, nil
}
