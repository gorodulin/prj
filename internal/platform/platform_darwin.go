//go:build darwin

package platform

import "os"

// ResolveLink resolves symlinks and macOS Finder Aliases.
// Tries symlink first (fast), then falls back to Finder alias resolution.
func ResolveLink(path string) (string, error) {
	target, err := os.Readlink(path)
	if err == nil {
		return target, nil
	}
	return ResolveAlias(path)
}

// SupportedLinkTypes returns link types available on macOS.
func SupportedLinkTypes() []string {
	return []string{"symlink", "finder-alias"}
}
