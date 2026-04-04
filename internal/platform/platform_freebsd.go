//go:build freebsd

package platform

import "os"

// ResolveLink resolves symlinks.
func ResolveLink(path string) (string, error) {
	return os.Readlink(path)
}

// SupportedLinkTypes returns link types available on FreeBSD.
func SupportedLinkTypes() []string {
	return []string{"symlink"}
}
