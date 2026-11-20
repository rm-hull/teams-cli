package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	"github.com/adrg/xdg"
	"github.com/gofrs/flock"
)

const (
	graphScope = "https://graph.microsoft.com/Chat.Read"
	cacheDir   = "teams-cli"
)

var aadCode = regexp.MustCompile(`\bAADSTS[0-9]+\b`)

func Authenticate(ctx context.Context, tenant, clientID string, login bool, prompts io.Writer) (string, error) {
	_, cachePath, lock, err := acquireCacheLock(tenant, clientID)
	if err != nil {
		return "", err
	}
	defer func() { _ = lock.Unlock() }()

	if !login {
		if _, err := os.Lstat(cachePath); errors.Is(err, os.ErrNotExist) {
			return "", errors.New("no cached sign-in; run login and complete sign-in in your browser")
		} else if err != nil {
			return "", fmt.Errorf("inspect token cache: %w", err)
		}
	}
	if _, err := readPrivateFileIfPresent(cachePath); err != nil {
		return "", err
	}

	app, err := public.New(clientID,
		public.WithAuthority("https://login.microsoftonline.com/"+tenant),
		public.WithCache(tokenCache{path: cachePath}),
	)
	if err != nil {
		return "", errors.New("could not initialize Entra sign-in; check the tenant ID or domain")
	}

	accounts, err := app.Accounts(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return "", context.Canceled
		}
		return "", errors.New("token cache is invalid; run logout, then login")
	}
	if login {
		for _, account := range accounts {
			if err := app.RemoveAccount(ctx, account); err != nil {
				if errors.Is(err, context.Canceled) {
					return "", context.Canceled
				}
				return "", errors.New("could not clear the previous account from the token cache")
			}
		}
		deviceCode, err := app.AcquireTokenByDeviceCode(ctx, []string{graphScope})
		if err != nil {
			return "", safeSignInError(err)
		}
		if deviceCode.Result.Message == "" || deviceCode.Result.UserCode == "" {
			return "", errors.New("Entra did not return valid device-code instructions") //nolint:staticcheck // Entra is a proper noun
		}
		if prompts != nil {
			if _, err := fmt.Fprintln(prompts, deviceCode.Result.Message); err != nil {
				return "", fmt.Errorf("write device-code instructions: %w", err)
			}
		}
		result, err := deviceCode.AuthenticationResult(ctx)
		if err != nil {
			return "", safeSignInError(err)
		}
		if result.AccessToken == "" {
			return "", errors.New("Entra sign-in returned no access token") //nolint:staticcheck // Entra is a proper noun
		}
		return result.AccessToken, nil
	}

	if len(accounts) != 1 {
		return "", errors.New("no single cached account; run login and complete browser sign-in")
	}
	result, err := app.AcquireTokenSilent(ctx, []string{graphScope}, public.WithSilentAccount(accounts[0]))
	if err != nil || result.AccessToken == "" {
		if errors.Is(err, context.Canceled) {
			return "", context.Canceled
		}
		return "", errors.New("cached sign-in is unavailable or expired; run login again")
	}
	return result.AccessToken, nil
}

func Logout(_ context.Context, tenant, clientID string) error {
	_, cachePath, lock, err := acquireCacheLock(tenant, clientID)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Unlock() }()
	if err := os.Remove(cachePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove token cache: %w", err)
	}
	return nil
}

func acquireCacheLock(tenant, clientID string) (string, string, *flock.Flock, error) {
	if !cachePlatformSupported() {
		return "", "", nil, errors.New("the token cache requires a POSIX system with owner-only permissions")
	}
	dir, err := xdg.StateFile(cacheDir)
	if err != nil {
		return "", "", nil, fmt.Errorf("determine state directory: %w", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", nil, fmt.Errorf("create token cache directory: %w", err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return "", "", nil, fmt.Errorf("inspect token cache directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 || !ownedByCurrentUser(info) {
		return "", "", nil, fmt.Errorf("token cache directory must be owner-only: %s", dir)
	}
	hash := sha256.Sum256([]byte(strings.ToLower(tenant) + ":" + strings.ToLower(clientID)))
	cachePath := filepath.Join(dir, fmt.Sprintf("%x.json", hash))
	lockPath := filepath.Join(dir, fmt.Sprintf("%x.lock", hash))

	if lockInfo, err := os.Lstat(lockPath); err == nil {
		if !lockInfo.Mode().IsRegular() || lockInfo.Mode().Perm()&0o077 != 0 || !ownedByCurrentUser(lockInfo) {
			return "", "", nil, errors.New("token cache lock must be an owner-only file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", "", nil, fmt.Errorf("inspect token cache lock: %w", err)
	}
	lock := flock.New(lockPath)
	ok, err := lock.TryLock()
	if err != nil {
		return "", "", nil, fmt.Errorf("lock token cache: %w", err)
	}
	if !ok {
		return "", "", nil, errors.New("another Teams command is accessing the token cache; try again")
	}
	if err := os.Chmod(lockPath, 0o600); err != nil {
		_ = lock.Unlock()
		return "", "", nil, fmt.Errorf("secure token cache lock: %w", err)
	}
	lockInfo, err := os.Lstat(lockPath)
	if err != nil || !lockInfo.Mode().IsRegular() || !ownedByCurrentUser(lockInfo) || lockInfo.Mode().Perm()&0o077 != 0 {
		_ = lock.Unlock()
		return "", "", nil, errors.New("token cache lock must be an owner-only file")
	}
	return dir, cachePath, lock, nil
}

func readPrivateFileIfPresent(path string) ([]byte, error) {
	data, err := readPrivateFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

func safeSignInError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if code := aadCode.FindString(err.Error()); code != "" {
		return fmt.Errorf("Entra sign-in failed (%s); check the app, tenant, and consent policy", code) //nolint:staticcheck // Entra is a proper noun
	}
	return errors.New("Entra sign-in failed; check the app, tenant, and consent policy") //nolint:staticcheck // Entra is a proper noun
}
