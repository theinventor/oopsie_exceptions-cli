package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theinventor/oopsie_exceptions-cli/internal/client"
)

type requestCapture struct {
	Method         string
	Path           string
	RawQuery       string
	Authorization  string
	ProjectID      string
	ContentType    string
	IdempotencyKey string
	Body           []byte
}

func runOopsie(t *testing.T, args []string, stdin string) (string, string, error) {
	t.Helper()
	root := NewRootCmd()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}

func withConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("OOPSIE_CONFIG", path)
	t.Setenv("OOPSIE_DISABLE_KEYCHAIN", "1")
	t.Setenv("OOPSIE_STORAGE", "file")
	t.Setenv(client.EnvAPIKey, "")
	t.Setenv(client.EnvAPIURL, "")
	t.Setenv(client.EnvProject, "")
	return path
}

func TestAuthSaveStatusAndListUseFileBackendWithoutLeakingSecret(t *testing.T) {
	cfg := withConfig(t)
	secret := "oopsie_secret_1234567890"

	stdout, _, err := runOopsie(t, []string{
		"auth", "save",
		"--profile", "prod",
		"--api-key", secret,
		"--api-url", "https://oopsie.example.com",
		"--storage", "file",
		"--project", "my-app",
	}, "")
	if err != nil {
		t.Fatalf("auth save: %v", err)
	}
	if strings.Contains(stdout, secret) {
		t.Fatalf("auth save leaked raw secret: %s", stdout)
	}
	if !strings.Contains(stdout, "oopsie_s...7890") {
		t.Fatalf("auth save should include masked fingerprint: %s", stdout)
	}
	info, err := os.Stat(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("config mode = %o, want 600", got)
	}

	status, _, err := runOopsie(t, []string{"auth", "status", "--profile", "prod"}, "")
	if err != nil {
		t.Fatalf("auth status: %v", err)
	}
	if strings.Contains(status, secret) {
		t.Fatalf("auth status leaked raw secret: %s", status)
	}
	if !strings.Contains(status, `"authenticated": true`) || !strings.Contains(status, `"name": "my-app"`) {
		t.Fatalf("auth status missing expected profile data: %s", status)
	}

	list, _, err := runOopsie(t, []string{"auth", "list"}, "")
	if err != nil {
		t.Fatalf("auth list: %v", err)
	}
	if strings.Contains(list, secret) {
		t.Fatalf("auth list leaked raw secret: %s", list)
	}
}

func TestErrorListSendsAuthProjectHeaderAndBoundedQuery(t *testing.T) {
	var cap requestCapture
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap = captureRequest(t, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"error_groups":[],"total":0}`)
	}))
	defer srv.Close()

	withConfig(t)
	t.Setenv(client.EnvAPIURL, srv.URL)
	t.Setenv(client.EnvAPIKey, "oopsie_env_key_123456")

	stdout, _, err := runOopsie(t, []string{
		"--project", "42",
		"error", "list",
		"--status", "unresolved",
		"--limit", "25",
		"--offset", "5",
	}, "")
	if err != nil {
		t.Fatalf("error list: %v stdout=%s", err, stdout)
	}
	if cap.Method != http.MethodGet || cap.Path != "/api/v1/error_groups" {
		t.Fatalf("request = %s %s", cap.Method, cap.Path)
	}
	if cap.Authorization != "Bearer oopsie_env_key_123456" {
		t.Fatalf("auth header = %q", cap.Authorization)
	}
	if cap.ProjectID != "42" {
		t.Fatalf("X-Project-Id = %q", cap.ProjectID)
	}
	if !strings.Contains(cap.RawQuery, "limit=25") || !strings.Contains(cap.RawQuery, "offset=5") || !strings.Contains(cap.RawQuery, "status=unresolved") {
		t.Fatalf("query = %q", cap.RawQuery)
	}
}

func TestProjectNameResolutionFetchesProjectsBeforeScopedRequest(t *testing.T) {
	var caps []requestCapture
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caps = append(caps, captureRequest(t, r))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/project":
			_, _ = io.WriteString(w, `{"projects":[{"id":7,"name":"my-app","error_groups_count":3,"unresolved_count":1,"created_at":"2026-05-20T00:00:00Z"}]}`)
		case "/api/v1/error_groups":
			_, _ = io.WriteString(w, `{"error_groups":[],"total":0}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	withConfig(t)
	t.Setenv(client.EnvAPIURL, srv.URL)
	t.Setenv(client.EnvAPIKey, "oopsie_env_key_123456")

	_, _, err := runOopsie(t, []string{"--project", "my-app", "error", "list"}, "")
	if err != nil {
		t.Fatalf("error list with project name: %v", err)
	}
	if len(caps) != 2 {
		t.Fatalf("expected project lookup + scoped request, got %d requests", len(caps))
	}
	if caps[0].Path != "/api/v1/project" || caps[1].Path != "/api/v1/error_groups" {
		t.Fatalf("paths = %#v", caps)
	}
	if caps[1].ProjectID != "7" {
		t.Fatalf("resolved project header = %q, want 7", caps[1].ProjectID)
	}
}

func TestNotificationCreateDryRunRedactsDestinationAndSkipsHTTP(t *testing.T) {
	withConfig(t)
	stdout, _, err := runOopsie(t, []string{
		"--project", "9",
		"notification", "create",
		"--url", "https://hooks.example.com/services/raw-secret-token",
		"--events", "error.created,error.reopened",
		"--dry-run",
	}, "")
	if err != nil {
		t.Fatalf("notification dry-run: %v", err)
	}
	if strings.Contains(stdout, "raw-secret-token") {
		t.Fatalf("dry-run leaked webhook secret: %s", stdout)
	}
	if !strings.Contains(stdout, `"destination_masked": "https://hooks.example.com/..."`) {
		t.Fatalf("dry-run missing masked destination: %s", stdout)
	}
	if !strings.Contains(stdout, `"project_id": "9"`) {
		t.Fatalf("dry-run missing project id: %s", stdout)
	}
}

func TestExceptionCreateFromPayloadFileSendsExpectedContract(t *testing.T) {
	var cap requestCapture
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap = captureRequest(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":1,"group_id":2,"is_new_group":true}`)
	}))
	defer srv.Close()

	withConfig(t)
	t.Setenv(client.EnvAPIURL, srv.URL)
	t.Setenv(client.EnvAPIKey, "oopsie_env_key_123456")
	payloadFile := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(payloadFile, []byte(`{"error":{"class_name":"RuntimeError","message":"boom","backtrace":["app.rb:1"]},"app":{"environment":"test"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := runOopsie(t, []string{
		"--project", "5",
		"exception", "create",
		"--payload-file", payloadFile,
		"--idempotency-key", "idem-123",
	}, "")
	if err != nil {
		t.Fatalf("exception create: %v stdout=%s", err, stdout)
	}
	if cap.Method != http.MethodPost || cap.Path != "/api/v1/exceptions" {
		t.Fatalf("request = %s %s", cap.Method, cap.Path)
	}
	if cap.ProjectID != "5" || cap.IdempotencyKey != "idem-123" {
		t.Fatalf("headers project=%q idem=%q", cap.ProjectID, cap.IdempotencyKey)
	}
	var body map[string]any
	if err := json.Unmarshal(cap.Body, &body); err != nil {
		t.Fatal(err)
	}
	errObj := body["error"].(map[string]any)
	if errObj["class_name"] != "RuntimeError" {
		t.Fatalf("body = %s", cap.Body)
	}
}

func TestAgentContextIsValidAndIncludesCoreCommands(t *testing.T) {
	withConfig(t)
	stdout, _, err := runOopsie(t, []string{"agent-context"}, "")
	if err != nil {
		t.Fatalf("agent-context: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	commands := parsed["commands"].(map[string]any)
	for _, name := range []string{"auth", "project", "error", "notification", "exception", "skill"} {
		if _, ok := commands[name]; !ok {
			t.Fatalf("agent-context missing command %q: %s", name, stdout)
		}
	}
}

func captureRequest(t *testing.T, r *http.Request) requestCapture {
	t.Helper()
	body, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()
	return requestCapture{
		Method:         r.Method,
		Path:           r.URL.Path,
		RawQuery:       r.URL.RawQuery,
		Authorization:  r.Header.Get("Authorization"),
		ProjectID:      r.Header.Get("X-Project-Id"),
		ContentType:    r.Header.Get("Content-Type"),
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
		Body:           body,
	}
}
