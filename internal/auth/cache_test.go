package auth

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testUnmarshaler struct {
	data []byte
}

func (u *testUnmarshaler) Unmarshal(data []byte) error {
	u.data = append([]byte(nil), data...)
	return nil
}

type testMarshaler []byte

func (m testMarshaler) Marshal() ([]byte, error) {
	return m, nil
}

func TestTokenCacheExportAndReplaceArePrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	provider := tokenCache{path: path}
	require.NoError(t, provider.Export(context.Background(), testMarshaler("serialized-cache"), cache.ExportHints{}))
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "cache permissions should be 0600")
	target := &testUnmarshaler{}
	require.NoError(t, provider.Replace(context.Background(), target, cache.ReplaceHints{}))
	assert.Equal(t, "serialized-cache", string(target.data))
}

func TestTokenCacheRejectsInsecureFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	require.NoError(t, os.WriteFile(path, []byte("cache"), 0o644))
	_, err := readPrivateFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "owner-only")
}

func TestCacheLockIsExclusiveAndRestrictive(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	_, _, lock, err := acquireCacheLock("tenant", "client")
	require.NoError(t, err)
	defer func() { _ = lock.Unlock() }()

	dir := filepath.Join(state, cacheDir)
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm(), "cache directory permissions should be 0700")
	_, _, second, err := acquireCacheLock("tenant", "client")
	if err == nil {
		_ = second.Unlock()
		require.Fail(t, "second cache lock unexpectedly succeeded")
	}
	assert.True(t, strings.Contains(err.Error(), "another Teams command"), "second cache lock error = %v", err)
}
