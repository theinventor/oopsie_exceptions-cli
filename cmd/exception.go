package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newExceptionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "exception",
		Short: "Create/report exception occurrences",
	}
	c.AddCommand(newExceptionCreateCmd())
	return c
}

func newExceptionCreateCmd() *cobra.Command {
	var payload, payloadFile string
	var fromStdin bool
	var className, message, appName, environment, timestamp, notifier, notifierVersion string
	var backtrace []string
	var backtraceFile, contextJSON, serverJSON string
	var handled bool
	var mf mutationFlags
	c := &cobra.Command{
		Use:     "create",
		Aliases: []string{"report"},
		Short:   "Report an exception to the scoped project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := resolveExceptionPayload(cmd, exceptionPayloadFlags{
				Payload:         payload,
				PayloadFile:     payloadFile,
				Stdin:           fromStdin,
				ClassName:       className,
				Message:         message,
				AppName:         appName,
				Environment:     environment,
				Timestamp:       timestamp,
				Notifier:        notifier,
				NotifierVersion: notifierVersion,
				Backtrace:       backtrace,
				BacktraceFile:   backtraceFile,
				ContextJSON:     contextJSON,
				ServerJSON:      serverJSON,
				Handled:         handled,
			})
			if err != nil {
				return err
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
				return printJSON(cmd.OutOrStdout(), newDryRunEnvelope(http.MethodPost, "/api/v1/exceptions", body, mf, projectID))
			}
			resp, err := cli.DoWithHeaders(http.MethodPost, "/api/v1/exceptions", body, nil, headers)
			return passthroughJSON(cmd, resp, err, http.MethodPost, "/api/v1/exceptions")
		},
	}
	c.Flags().StringVar(&payload, "payload", "", "raw ExceptionReporter JSON payload")
	c.Flags().StringVar(&payloadFile, "payload-file", "", "read raw ExceptionReporter JSON payload from file")
	c.Flags().BoolVar(&fromStdin, "stdin", false, "read raw ExceptionReporter JSON payload from stdin")
	c.Flags().StringVar(&className, "class", "", "exception class name when building a payload")
	c.Flags().StringVar(&message, "message", "", "exception message when building a payload")
	c.Flags().StringArrayVar(&backtrace, "backtrace", nil, "backtrace line; repeat for multiple lines")
	c.Flags().StringVar(&backtraceFile, "backtrace-file", "", "read backtrace lines from a file")
	c.Flags().StringVar(&appName, "app", "", "application name")
	c.Flags().StringVar(&environment, "environment", "", "application environment")
	c.Flags().StringVar(&timestamp, "timestamp", "", "occurrence timestamp (RFC3339); defaults to now")
	c.Flags().BoolVar(&handled, "handled", false, "mark the exception as handled")
	c.Flags().StringVar(&contextJSON, "context-json", "", "JSON object for payload.context")
	c.Flags().StringVar(&serverJSON, "server-json", "", "JSON object for payload.server")
	c.Flags().StringVar(&notifier, "notifier", "oopsie-cli", "payload notifier name")
	c.Flags().StringVar(&notifierVersion, "notifier-version", Version, "payload notifier version")
	bindMutationFlags(c, &mf)
	return c
}

type exceptionPayloadFlags struct {
	Payload         string
	PayloadFile     string
	Stdin           bool
	ClassName       string
	Message         string
	AppName         string
	Environment     string
	Timestamp       string
	Notifier        string
	NotifierVersion string
	Backtrace       []string
	BacktraceFile   string
	ContextJSON     string
	ServerJSON      string
	Handled         bool
}

func resolveExceptionPayload(cmd *cobra.Command, flags exceptionPayloadFlags) (map[string]any, error) {
	rawSources := 0
	if flags.Payload != "" {
		rawSources++
	}
	if flags.PayloadFile != "" {
		rawSources++
	}
	if flags.Stdin {
		rawSources++
	}
	if rawSources > 1 {
		return nil, usageError("use only one of --payload, --payload-file, or --stdin")
	}
	if rawSources == 1 {
		var raw []byte
		var err error
		switch {
		case flags.Payload != "":
			raw = []byte(flags.Payload)
		case flags.PayloadFile != "":
			raw, err = os.ReadFile(flags.PayloadFile)
		case flags.Stdin:
			raw, err = io.ReadAll(cmd.InOrStdin())
		}
		if err != nil {
			return nil, usageError("read payload: %v", err)
		}
		var body map[string]any
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		if err := dec.Decode(&body); err != nil {
			return nil, usageError("payload must be a JSON object: %v", err)
		}
		if _, ok := body["error"].(map[string]any); !ok {
			return nil, usageError("payload must include an error object")
		}
		return body, nil
	}

	if flags.ClassName == "" {
		return nil, usageError("--class is required when not using --payload, --payload-file, or --stdin")
	}
	ts := flags.Timestamp
	if ts == "" {
		ts = time.Now().UTC().Format(time.RFC3339Nano)
	}
	lines := append([]string{}, flags.Backtrace...)
	if flags.BacktraceFile != "" {
		file, err := os.Open(flags.BacktraceFile)
		if err != nil {
			return nil, usageError("read --backtrace-file: %v", err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if line := strings.TrimSpace(scanner.Text()); line != "" {
				lines = append(lines, line)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, usageError("read --backtrace-file: %v", err)
		}
	}
	context, err := parseJSONObject(flags.ContextJSON, "context-json")
	if err != nil {
		return nil, err
	}
	server, err := parseJSONObject(flags.ServerJSON, "server-json")
	if err != nil {
		return nil, err
	}
	app := map[string]any{}
	if flags.AppName != "" {
		app["name"] = flags.AppName
	}
	if flags.Environment != "" {
		app["environment"] = flags.Environment
	}
	body := map[string]any{
		"notifier":  flags.Notifier,
		"version":   flags.NotifierVersion,
		"timestamp": ts,
		"error": map[string]any{
			"class_name": flags.ClassName,
			"message":    flags.Message,
			"backtrace":  lines,
			"handled":    flags.Handled,
		},
	}
	if len(app) > 0 {
		body["app"] = app
	}
	if context != nil {
		body["context"] = context
	}
	if server != nil {
		body["server"] = server
	}
	return body, nil
}
