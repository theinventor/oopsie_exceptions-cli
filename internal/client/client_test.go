package client

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/theinventor/oopsie_exceptions-cli/internal/config"
)

func TestExplicitProfileWinsAndMissingProfileDoesNotFallBackToEnv(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("OOPSIE_CONFIG", cfg)
	t.Setenv(EnvAPIKey, "ENV_KEY_SHOULD_NOT_WIN")
	t.Setenv(EnvAPIURL, "https://env.example.com")

	f := &config.File{}
	f.Put("prod", config.Profile{
		APIURL: "https://prod.example.com",
		APIKey: "PROFILE_KEY_LONG_ENOUGH",
	})
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}

	c := NewWithOptions("prod", "")
	if c.APIKey != "PROFILE_KEY_LONG_ENOUGH" || c.BaseURL != "https://prod.example.com" {
		t.Fatalf("explicit profile not used: key=%q url=%q", c.APIKey, c.BaseURL)
	}

	missing := NewWithOptions("missing", "")
	if missing.APIKey != "" {
		t.Fatalf("missing explicit profile fell back to env key: %q", missing.APIKey)
	}
	if missing.BaseURL != "https://env.example.com" {
		t.Fatalf("missing explicit profile should keep URL diagnostics, got %q", missing.BaseURL)
	}
}

func TestEnvWinsOverDefaultProfileWhenProfileNotExplicit(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("OOPSIE_CONFIG", cfg)
	t.Setenv(EnvAPIKey, "ENV_KEY_LONG_ENOUGH")
	t.Setenv(EnvAPIURL, "https://env.example.com")

	f := &config.File{}
	f.Put("prod", config.Profile{APIURL: "https://prod.example.com", APIKey: "PROFILE_KEY"})
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}

	c := New()
	if c.APIKey != "ENV_KEY_LONG_ENOUGH" || c.BaseURL != "https://env.example.com" || c.Source != "env" {
		t.Fatalf("env did not win: %#v", c)
	}
}

func TestMaskKeyNeverReturnsRawShortSecret(t *testing.T) {
	if got := MaskKey("short"); got != "***" {
		t.Fatalf("short secret mask = %q", got)
	}
	if got := MaskKey(""); got != "(none)" {
		t.Fatalf("empty secret mask = %q", got)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
