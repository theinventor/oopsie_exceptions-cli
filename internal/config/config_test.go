package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveCreatesMode0600ConfigAndStableNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("OOPSIE_CONFIG", path)

	f := &File{}
	f.Put("b", Profile{APIURL: "https://b.example.com/"})
	f.Put("a", Profile{APIURL: "https://a.example.com"})
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Names(); got[0] != "a" || got[1] != "b" {
		t.Fatalf("names not sorted: %#v", got)
	}
	if loaded.Profiles["b"].APIURL != "https://b.example.com" {
		t.Fatalf("api_url not normalized: %#v", loaded.Profiles["b"])
	}
	raw, _ := os.ReadFile(path)
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
}
