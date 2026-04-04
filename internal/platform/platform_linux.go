//go:build linux

package platform

import "os"

// ResolveLink resolves symlinks.
func ResolveLink(path string) (string, error) {
	return os.Readlink(path)
}

// SupportedLinkTypes returns link types available on Linux.
func SupportedLinkTypes() []string {
	return []string{"symlink"}
}
