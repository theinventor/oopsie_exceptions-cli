package cmd

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/theinventor/oopsie_exceptions-cli/internal/enums"
)

func newErrorCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "error",
		Short: "List, inspect, and transition Oopsie error groups",
	}
	c.AddCommand(newErrorListCmd())
	c.AddCommand(newErrorGetCmd())
	c.AddCommand(newErrorMutationCmd("resolve", "resolve", "Mark an error group resolved"))
	c.AddCommand(newErrorMutationCmd("ignore", "ignore", "Ignore/archive an error group"))
	c.AddCommand(newErrorMutationCmd("reopen", "unresolve", "Reopen an error group"))
	return c
}

func newErrorListCmd() *cobra.Command {
	var status, limitRaw, offsetRaw string
	c := &cobra.Command{
		Use:   "list",
		Short: "List error groups for the scoped project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := enums.Validate("status", status, enums.ErrorStatuses); err != nil {
				return usageError("%v", err)
			}
			limit, err := parseBoundedInt("limit", limitRaw, 50, 1, 100)
			if err != nil {
				return err
			}
			offset, err := parseBoundedInt("offset", offsetRaw, 0, 0, 10000)
			if err != nil {
				return err
			}
			query := url.Values{}
			query.Set("limit", intString(limit))
			query.Set("offset", intString(offset))
			if status != "" {
				query.Set("status", status)
			}
			cli := newAPIClient()
			headers, _, err := projectHeaders(cmd, cli)
			if err != nil {
				return err
			}
			resp, err := cli.DoWithHeaders(http.MethodGet, "/api/v1/error_groups", nil, query, headers)
			return passthroughJSON(cmd, resp, err, http.MethodGet, "/api/v1/error_groups")
		},
	}
	c.Flags().StringVar(&status, "status", "", "filter by status: unresolved, resolved, or ignored")
	c.Flags().StringVar(&limitRaw, "limit", "50", "maximum groups to return (1-100)")
	c.Flags().StringVar(&offsetRaw, "offset", "0", "offset for pagination (0-10000)")
	return c
}

func newErrorGetCmd() *cobra.Command {
	var limitRaw string
	c := &cobra.Command{
		Use:   "get <error-group-id>",
		Short: "Show an error group and recent occurrences",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, err := parseBoundedInt("limit", limitRaw, 20, 1, 50)
			if err != nil {
				return err
			}
			query := url.Values{"limit": []string{intString(limit)}}
			cli := newAPIClient()
			headers, _, err := projectHeaders(cmd, cli)
			if err != nil {
				return err
			}
			path := "/api/v1/error_groups/" + args[0]
			resp, err := cli.DoWithHeaders(http.MethodGet, path, nil, query, headers)
			return passthroughJSON(cmd, resp, err, http.MethodGet, path)
		},
	}
	c.Flags().StringVar(&limitRaw, "limit", "20", "maximum recent occurrences to return (1-50)")
	return c
}

func newErrorMutationCmd(name, apiAction, short string) *cobra.Command {
	var mf mutationFlags
	c := &cobra.Command{
		Use:   name + " <error-group-id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runErrorMutation(cmd, args[0], apiAction, mf)
		},
	}
	bindMutationFlags(c, &mf)
	return c
}

func runErrorMutation(cmd *cobra.Command, id, apiAction string, mf mutationFlags) error {
	cli := newAPIClient()
	headers, projectID, err := projectHeaders(cmd, cli)
	if err != nil {
		return err
	}
	for k, v := range mf.Headers() {
		if headers == nil {
			headers = map[string]string{}
		}
		headers[k] = v
	}
	path := "/api/v1/error_groups/" + id + "/" + apiAction
	if mf.DryRun {
		return printJSON(cmd.OutOrStdout(), newDryRunEnvelope(http.MethodPatch, path, nil, mf, projectID))
	}
	resp, err := cli.DoWithHeaders(http.MethodPatch, path, nil, nil, headers)
	return passthroughJSON(cmd, resp, err, http.MethodPatch, path)
}

func newErrorsAliasCmd() *cobra.Command {
	cmd := newErrorListCmd()
	cmd.Use = "errors"
	cmd.Hidden = true
	cmd.Deprecated = "use `oopsie error list`"
	return cmd
}

func newShowAliasCmd() *cobra.Command {
	cmd := newErrorGetCmd()
	cmd.Use = "show <error-group-id>"
	cmd.Hidden = true
	cmd.Deprecated = "use `oopsie error get <error-group-id>`"
	return cmd
}

func newErrorMutationAliasCmd(name, apiAction string) *cobra.Command {
	cmd := newErrorMutationCmd(name, apiAction, "Deprecated alias")
	cmd.Hidden = true
	cmd.Deprecated = "use `oopsie error " + name + " <error-group-id>`"
	return cmd
}

func intString(n int) string {
	return strconv.Itoa(n)
}
