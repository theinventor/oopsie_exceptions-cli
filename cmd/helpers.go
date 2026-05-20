package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theinventor/oopsie_exceptions-cli/internal/exitcode"
)

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func readAPIResponse(cmd *cobra.Command, resp *http.Response, err error, method, path string) ([]byte, error) {
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Network, fmt.Errorf("%s %s: %w", method, path, err))
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, exitcode.Wrap(exitcode.Network, fmt.Errorf("read response: %w", readErr))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if len(bytes.TrimSpace(body)) > 0 {
			_ = writeBody(cmd.OutOrStdout(), body)
		}
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = "empty response body; check that --api-url points at an Oopsie API server"
		}
		return nil, exitcode.Wrap(exitcode.FromHTTPStatus(resp.StatusCode),
			fmt.Errorf("HTTP %d from %s %s: %s", resp.StatusCode, method, path, msg))
	}
	return body, nil
}

func passthroughJSON(cmd *cobra.Command, resp *http.Response, err error, method, path string) error {
	body, readErr := readAPIResponse(cmd, resp, err, method, path)
	if readErr != nil {
		return readErr
	}
	return writeBody(cmd.OutOrStdout(), body)
}

func writeBody(w io.Writer, body []byte) error {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		_, err := fmt.Fprintln(w, "{}")
		return err
	}
	var pretty bytes.Buffer
	if json.Valid(body) {
		if err := json.Indent(&pretty, body, "", "  "); err == nil {
			_, err = pretty.WriteTo(w)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(w)
			return err
		}
	}
	_, err := fmt.Fprintln(w, string(body))
	return err
}

func runAPIJSON(cmd *cobra.Command, method, path string, body any, query url.Values, headers map[string]string) error {
	cli := newAPIClient()
	resp, err := cli.DoWithHeaders(method, path, body, query, headers)
	return passthroughJSON(cmd, resp, err, method, path)
}

func usageError(format string, args ...any) error {
	return exitcode.Wrap(exitcode.Usage, fmt.Errorf(format, args...))
}

func parseBoundedInt(name, raw string, def, min, max int) (int, error) {
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, usageError("--%s must be an integer", name)
	}
	if n < min || n > max {
		return 0, usageError("--%s must be between %d and %d", name, min, max)
	}
	return n, nil
}

func isNumericID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func readTrimmedFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

func parseJSONObject(raw, flag string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var out map[string]any
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		return nil, usageError("--%s must be a JSON object: %v", flag, err)
	}
	return out, nil
}

func maskDestination(channel, destination string) string {
	if destination == "" {
		return ""
	}
	if channel == "webhook" {
		u, err := url.Parse(destination)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return "[masked]"
		}
		return u.Scheme + "://" + u.Host + "/..."
	}
	local, domain, ok := strings.Cut(destination, "@")
	if !ok || local == "" || domain == "" {
		return "[masked]"
	}
	return local[:1] + "***@" + domain
}
