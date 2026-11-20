package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
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
	if err := provider.Export(context.Background(), testMarshaler("serialized-cache"), cache.ExportHints{}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("cache permissions = %04o, want 0600", info.Mode().Perm())
	}
	target := &testUnmarshaler{}
	if err := provider.Replace(context.Background(), target, cache.ReplaceHints{}); err != nil {
		t.Fatal(err)
	}
	if string(target.data) != "serialized-cache" {
		t.Fatalf("cache data = %q", target.data)
	}
}

func TestTokenCacheRejectsInsecureFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	if err := os.WriteFile(path, []byte("cache"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readPrivateFile(path)
	if err == nil || !strings.Contains(err.Error(), "owner-only") {
		t.Fatalf("readPrivateFile error = %v, want owner-only permission error", err)
	}
}

func TestCacheLockIsExclusiveAndRestrictive(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	_, _, lock, err := acquireCacheLock("tenant", "client")
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Unlock()

	dir := filepath.Join(state, cacheDir)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("cache directory permissions = %04o, want 0700", info.Mode().Perm())
	}
	_, _, second, err := acquireCacheLock("tenant", "client")
	if err == nil {
		_ = second.Unlock()
		t.Fatal("second cache lock unexpectedly succeeded")
	}
	if !errors.Is(err, os.ErrExist) && !strings.Contains(err.Error(), "another Teams command") {
		t.Fatalf("second cache lock error = %v", err)
	}
}
