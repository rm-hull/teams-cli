package config

import (
	"os"
	"path/filepath"
	"testing"
)

const testClientID = "00001111-aaaa-2222-bbbb-3333cccc4444"

func TestLoadPrioritiesAndDotEnvToken(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(
		"TEAMS_TENANT_ID=file.onmicrosoft.com\nTEAMS_CLIENT_ID="+testClientID+"\nACCESS_TOKEN=raw-token\n",
	), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEAMS_ENV_FILE", filepath.Join(dir, ".env"))
	t.Setenv("TEAMS_TENANT_ID", "env.onmicrosoft.com")
	settings, err := Load("flag.onmicrosoft.com", "")
	if err != nil {
		t.Fatal(err)
	}
	if settings.Tenant != "flag.onmicrosoft.com" || settings.ClientID != testClientID || settings.AccessToken != "raw-token" {
		t.Fatalf("unexpected settings: %#v", settings)
	}
}

func TestLoadRejectsBearerPrefixWithoutEchoingToken(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("ACCESS_TOKEN=Bearer sensitive\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEAMS_ENV_FILE", filepath.Join(dir, ".env"))
	t.Setenv("TEAMS_TENANT_ID", "example.onmicrosoft.com")
	t.Setenv("TEAMS_CLIENT_ID", testClientID)
	_, err := Load("", "")
	if err == nil || !contains(err.Error(), "without the Bearer prefix") || contains(err.Error(), "sensitive") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDotEnvTokenDoesNotRequireAppConfiguration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("ACCESS_TOKEN=raw-token\nTEAMS_TENANT_ID=https://invalid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEAMS_ENV_FILE", path)
	settings, err := Load("", "")
	if err != nil {
		t.Fatal(err)
	}
	if settings.AccessToken != "raw-token" {
		t.Fatalf("AccessToken = %q, want raw-token", settings.AccessToken)
	}
	if err := settings.RequireApplication(); err == nil {
		t.Fatal("RequireApplication unexpectedly succeeded")
	}
}

func contains(value, substr string) bool {
	for i := 0; i+len(substr) <= len(value); i++ {
		if value[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
