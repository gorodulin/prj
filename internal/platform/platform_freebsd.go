//go:build freebsd

package platform

import "os"

// ResolveLink resolves symlinks.
// Returns (target, kind, error) where kind is "symlink".
func ResolveLink(path string) (string, string, error) {
	target, err := os.Readlink(path)
	if err != nil {
		return "", "", err
	}
	return target, "symlink", nil
}

// SupportedLinkTypes returns link types available on FreeBSD.
func SupportedLinkTypes() []string {
	return []string{"symlink"}
}

// DefaultLinkKind returns the link kind to use when none is configured.
func DefaultLinkKind() string {
	return "symlink"
}
