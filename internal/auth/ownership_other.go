//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package auth

import "os"

func ownedByCurrentUser(os.FileInfo) bool {
	return false
}

func cachePlatformSupported() bool {
	return false
}
