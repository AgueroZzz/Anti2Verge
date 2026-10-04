//go:build !windows

package winutil

import "errors"

func QueryAppPathFromRegistry(appName string) (string, error) {
	return "", errors.New("registry not supported on this platform")
}

func QueryUninstallDisplayName(targetName string) (string, error) {
	return "", errors.New("registry not supported on this platform")
}
