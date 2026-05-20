package cmd

import (
	"runtime/debug"

	"github.com/spf13/cobra"
	"github.com/theinventor/oopsie_exceptions-cli/internal/client"
)

var Version = "dev"
var Commit = ""
var BuildDate = ""

var rootProfile string
var rootProject string
var rootAPIURL string

func init() {
	if Version != "dev" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}
}

func NewRootCmd() *cobra.Command {
	rootProfile = ""
	rootProject = ""
	rootAPIURL = ""

	root := &cobra.Command{
		Use:           "oopsie",
		Short:         "JSON-first CLI for Oopsie exception tracking",
		Long:          "oopsie is a JSON-first CLI for self-hosted Oopsie exception trackers. Configure with OOPSIE_API_KEY and OOPSIE_API_URL, or save a profile with `oopsie auth save`.",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&rootProfile, "profile", "", "saved auth profile to use for this invocation")
	root.PersistentFlags().StringVarP(&rootProject, "project", "p", "", "project id or exact project name for project-scoped endpoints")
	root.PersistentFlags().StringVar(&rootAPIURL, "api-url", "", "Oopsie API base URL for this invocation")

	root.AddCommand(newAgentContextCmd())
	root.AddCommand(newAuthCmd())
	root.AddCommand(newWhoamiCmd())
	root.AddCommand(newProjectCmd())
	root.AddCommand(newErrorCmd())
	root.AddCommand(newNotificationCmd())
	root.AddCommand(newExceptionCmd())
	root.AddCommand(newSkillCmd())
	root.AddCommand(newVersionCmd())

	root.AddCommand(newProjectsAliasCmd())
	root.AddCommand(newErrorsAliasCmd())
	root.AddCommand(newShowAliasCmd())
	root.AddCommand(newErrorMutationAliasCmd("resolve", "resolve"))
	root.AddCommand(newErrorMutationAliasCmd("ignore", "ignore"))
	root.AddCommand(newErrorMutationAliasCmd("reopen", "unresolve"))
	root.AddCommand(newNotificationsAliasCmd())
	root.AddCommand(newWebhookAliasCmd())
	root.AddCommand(newWebhooksAliasCmd())
	root.AddCommand(newConfigAliasCmd())

	return root
}

func newAPIClient() *client.Client {
	c := client.NewWithOptions(rootProfile, rootAPIURL)
	c.Version = Version
	return c
}
