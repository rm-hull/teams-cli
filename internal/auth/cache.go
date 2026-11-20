package auth

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

type tokenCache struct {
	path string
}

func (c tokenCache) Replace(_ context.Context, target cache.Unmarshaler, _ cache.ReplaceHints) error {
	data, err := readPrivateFile(c.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := target.Unmarshal(data); err != nil {
		return fmt.Errorf("invalid cache data: %w", err)
	}
	return nil
}

func (c tokenCache) Export(_ context.Context, source cache.Marshaler, _ cache.ExportHints) error {
	data, err := source.Marshal()
	if err != nil {
		return fmt.Errorf("serialize token cache: %w", err)
	}
	return writePrivateFile(c.path, data)
}

func readPrivateFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || !ownedByCurrentUser(info) {
		return nil, errors.New("token cache must be an owner-only file; run logout, then login")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read token cache: %w", err)
	}
	return data, nil
}

func writePrivateFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, ".cache-*")
	if err != nil {
		return fmt.Errorf("create token cache file: %w", err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)

	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return fmt.Errorf("secure token cache file: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write token cache: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync token cache: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close token cache: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace token cache: %w", err)
	}
	directory, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open token cache directory: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync token cache directory: %w", err)
	}
	return nil
}
