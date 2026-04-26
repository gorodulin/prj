//go:build darwin

package platform

import "os"

// ResolveLink resolves symlinks and macOS Finder Aliases.
// Tries symlink first (fast), then falls back to Finder alias resolution.
// Returns (target, kind, error) where kind is "symlink" or "finder-alias".
func ResolveLink(path string) (string, string, error) {
	target, err := os.Readlink(path)
	if err == nil {
		return target, "symlink", nil
	}
	target, err = ResolveAlias(path)
	if err != nil {
		return "", "", err
	}
	return target, "finder-alias", nil
}

// SupportedLinkTypes returns link types available on macOS.
func SupportedLinkTypes() []string {
	return []string{"symlink", "finder-alias"}
}

// DefaultLinkKind returns the link kind to use when none is configured.
func DefaultLinkKind() string {
	return "symlink"
}
