package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theinventor/oopsie_exceptions-cli/internal/enums"
)

func newNotificationCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "notification",
		Short: "List and create notification rules",
	}
	c.AddCommand(newNotificationListCmd())
	c.AddCommand(newNotificationCreateCmd())
	return c
}

func newNotificationListCmd() *cobra.Command {
	var channel string
	c := &cobra.Command{
		Use:   "list",
		Short: "List notification rules for the scoped project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := enums.Validate("channel", channel, enums.NotificationChannels); err != nil {
				return usageError("%v", err)
			}
			cli := newAPIClient()
			headers, _, err := projectHeaders(cmd, cli)
			if err != nil {
				return err
			}
			resp, err := cli.DoWithHeaders(http.MethodGet, "/api/v1/notification_rules", nil, nil, headers)
			if channel == "" {
				return passthroughJSON(cmd, resp, err, http.MethodGet, "/api/v1/notification_rules")
			}
			body, readErr := readAPIResponse(cmd, resp, err, http.MethodGet, "/api/v1/notification_rules")
			if readErr != nil {
				return readErr
			}
			var parsed struct {
				NotificationRules []map[string]any `json:"notification_rules"`
			}
			if err := jsonUnmarshal(body, &parsed); err != nil {
				return err
			}
			filtered := []map[string]any{}
			for _, rule := range parsed.NotificationRules {
				if rule["channel"] == channel {
					filtered = append(filtered, rule)
				}
			}
			return printJSON(cmd.OutOrStdout(), map[string]any{"notification_rules": filtered})
		},
	}
	c.Flags().StringVar(&channel, "channel", "", "filter by notification channel: email or webhook")
	c.Flags().StringVar(&channel, "kind", "", "deprecated alias for --channel")
	return c
}

func newNotificationCreateCmd() *cobra.Command {
	var channel, destination, urlValue, urlFile, eventsRaw string
	var urlStdin, enabled, disabled bool
	var mf mutationFlags
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a notification rule for the scoped project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if disabled {
				enabled = false
			}
			if channel == "" {
				channel = "webhook"
			}
			if err := enums.Validate("channel", channel, enums.NotificationChannels); err != nil {
				return usageError("%v", err)
			}
			dest, err := resolveDestination(cmd, destination, urlValue, urlFile, urlStdin)
			if err != nil {
				return err
			}
			if channel == "webhook" {
				if _, err := url.ParseRequestURI(dest); err != nil || !(strings.HasPrefix(dest, "http://") || strings.HasPrefix(dest, "https://")) {
					return usageError("webhook destination must be an HTTP or HTTPS URL")
				}
			}
			events, err := parseNotificationEvents(eventsRaw)
			if err != nil {
				return err
			}
			body := map[string]any{
				"notification_rule": map[string]any{
					"channel":     channel,
					"destination": dest,
					"events":      events,
					"enabled":     enabled,
				},
			}
			redactedBody := map[string]any{
				"notification_rule": map[string]any{
					"channel":              channel,
					"destination_masked":   maskDestination(channel, dest),
					"destination_redacted": true,
					"events":               events,
					"enabled":              enabled,
				},
			}
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
			if mf.DryRun {
				return printJSON(cmd.OutOrStdout(), newDryRunEnvelope(http.MethodPost, "/api/v1/notification_rules", redactedBody, mf, projectID))
			}
			resp, err := cli.DoWithHeaders(http.MethodPost, "/api/v1/notification_rules", body, nil, headers)
			return passthroughJSON(cmd, resp, err, http.MethodPost, "/api/v1/notification_rules")
		},
	}
	c.Flags().StringVar(&channel, "channel", "", "notification channel: email or webhook (default webhook)")
	c.Flags().StringVar(&channel, "kind", "", "deprecated alias for --channel")
	c.Flags().StringVar(&destination, "destination", "", "notification destination")
	c.Flags().StringVar(&urlValue, "url", "", "webhook URL alias for --destination")
	c.Flags().StringVar(&urlFile, "url-file", "", "read webhook URL from this file")
	c.Flags().BoolVar(&urlStdin, "url-stdin", false, "read webhook URL from stdin")
	c.Flags().StringVar(&eventsRaw, "events", "new_error,regression", "comma-separated events: new_error, regression, error.created, error.reopened")
	c.Flags().StringVar(&eventsRaw, "event", "new_error,regression", "deprecated alias for --events")
	c.Flags().BoolVar(&enabled, "enabled", true, "create the rule enabled")
	c.Flags().BoolVar(&disabled, "disabled", false, "create the rule disabled")
	bindMutationFlags(c, &mf)
	return c
}

func resolveDestination(cmd *cobra.Command, destination, urlValue, urlFile string, urlStdin bool) (string, error) {
	values := []string{}
	if destination != "" {
		values = append(values, destination)
	}
	if urlValue != "" {
		values = append(values, urlValue)
	}
	if urlFile != "" {
		v, err := readTrimmedFile(urlFile)
		if err != nil {
			return "", usageError("read --url-file: %v", err)
		}
		values = append(values, v)
	}
	if urlStdin {
		raw, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", usageError("read --url-stdin: %v", err)
		}
		values = append(values, strings.TrimSpace(string(raw)))
	}
	if len(values) == 0 {
		return "", usageError("missing destination; use --destination, --url, --url-file, or --url-stdin")
	}
	if len(values) > 1 {
		return "", usageError("use only one destination source")
	}
	return values[0], nil
}

func parseNotificationEvents(raw string) ([]string, error) {
	seen := map[string]bool{}
	events := []string{}
	for _, part := range strings.Split(raw, ",") {
		event, ok := enums.CanonicalNotificationEvent(part)
		if !ok {
			return nil, usageError("invalid notification event %q; supported: %s", strings.TrimSpace(part), strings.Join(enums.InContext["notification_event"], ", "))
		}
		if !seen[event] {
			seen[event] = true
			events = append(events, event)
		}
	}
	if len(events) == 0 {
		return nil, usageError("--events must include at least one event")
	}
	return events, nil
}

func newNotificationsAliasCmd() *cobra.Command {
	cmd := newNotificationListCmd()
	cmd.Use = "notifications"
	cmd.Hidden = true
	cmd.Deprecated = "use `oopsie notification list`"
	return cmd
}

func newWebhooksAliasCmd() *cobra.Command {
	cmd := newNotificationListCmd()
	cmd.Use = "webhooks"
	cmd.Hidden = true
	cmd.Deprecated = "use `oopsie notification list --channel webhook`"
	cmd.PreRun = func(cmd *cobra.Command, _ []string) {
		_ = cmd.Flags().Set("channel", "webhook")
	}
	return cmd
}

func newWebhookAliasCmd() *cobra.Command {
	c := &cobra.Command{
		Use:        "webhook",
		Short:      "Deprecated notification webhook alias",
		Hidden:     true,
		Deprecated: "use `oopsie notification ...`",
	}
	list := newNotificationListCmd()
	list.Use = "list"
	list.PreRun = func(cmd *cobra.Command, _ []string) { _ = cmd.Flags().Set("channel", "webhook") }
	create := newNotificationCreateCmd()
	create.Use = "create"
	create.PreRun = func(cmd *cobra.Command, _ []string) { _ = cmd.Flags().Set("channel", "webhook") }
	c.AddCommand(list, create)
	return c
}

func jsonUnmarshal(body []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("parse server JSON: %w", err)
	}
	return nil
}
