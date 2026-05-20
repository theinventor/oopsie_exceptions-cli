package credstore

import "testing"

func TestFileAndKeychainBackendsRoundTrip(t *testing.T) {
	_, restore := UseMockKeyring()
	defer restore()

	if backend, err := Put("prod", BackendFile, "secret"); err != nil || backend != BackendFile {
		t.Fatalf("file Put backend=%q err=%v", backend, err)
	}
	if got, err := Get("prod", BackendFile, "secret"); err != nil || got != "secret" {
		t.Fatalf("file Get got=%q err=%v", got, err)
	}

	if backend, err := Put("prod", BackendKeychain, "secret2"); err != nil || backend != BackendKeychain {
		t.Fatalf("keychain Put backend=%q err=%v", backend, err)
	}
	if got, err := Get("prod", BackendKeychain, ""); err != nil || got != "secret2" {
		t.Fatalf("keychain Get got=%q err=%v", got, err)
	}
	if err := Delete("prod", BackendKeychain); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := Get("prod", BackendKeychain, ""); err != ErrNotFound {
		t.Fatalf("after delete err=%v, want ErrNotFound", err)
	}
}

func TestDeleteSkipsKeychainForFileBackends(t *testing.T) {
	k, restore := UseMockKeyring()
	defer restore()
	k.FailDelete = true

	if err := Delete("prod", BackendFile); err != nil {
		t.Fatalf("file delete should not touch keychain: %v", err)
	}
	if err := Delete("legacy", ""); err != nil {
		t.Fatalf("legacy file delete should not touch keychain: %v", err)
	}
}

func TestResolveBackendHonorsDisableKeychain(t *testing.T) {
	_, restore := UseMockKeyring()
	defer restore()
	t.Setenv(EnvDisableKeychain, "1")

	got, err := ResolveBackend(BackendAuto)
	if err != nil {
		t.Fatal(err)
	}
	if got != BackendFile {
		t.Fatalf("auto with disabled keychain = %q, want file", got)
	}
}
