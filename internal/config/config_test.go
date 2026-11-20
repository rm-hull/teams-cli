package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testClientID = "00001111-aaaa-2222-bbbb-3333cccc4444"

func TestLoadPrioritiesAndDotEnvToken(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(
		"TEAMS_TENANT_ID=file.onmicrosoft.com\nTEAMS_CLIENT_ID="+testClientID+"\nACCESS_TOKEN=raw-token\n",
	), 0o600))
	t.Setenv("TEAMS_ENV_FILE", filepath.Join(dir, ".env"))
	t.Setenv("TEAMS_TENANT_ID", "env.onmicrosoft.com")
	settings, err := Load("flag.onmicrosoft.com", "")
	require.NoError(t, err)
	assert.Equal(t, "flag.onmicrosoft.com", settings.Tenant)
	assert.Equal(t, testClientID, settings.ClientID)
	assert.Equal(t, "raw-token", settings.AccessToken)
}

func TestLoadRejectsBearerPrefixWithoutEchoingToken(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("ACCESS_TOKEN=Bearer sensitive\n"), 0o600))
	t.Setenv("TEAMS_ENV_FILE", filepath.Join(dir, ".env"))
	t.Setenv("TEAMS_TENANT_ID", "example.onmicrosoft.com")
	t.Setenv("TEAMS_CLIENT_ID", testClientID)
	_, err := Load("", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "without the Bearer prefix")
	assert.NotContains(t, err.Error(), "sensitive")
}

func TestDotEnvTokenDoesNotRequireAppConfiguration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte("ACCESS_TOKEN=raw-token\nTEAMS_TENANT_ID=https://invalid\n"), 0o600))
	t.Setenv("TEAMS_ENV_FILE", path)
	settings, err := Load("", "")
	require.NoError(t, err)
	assert.Equal(t, "raw-token", settings.AccessToken)
	assert.Error(t, settings.RequireApplication())
}
