//go:build windows

package platform

import "os"

// ResolveLink resolves symlinks and NTFS junctions.
// TODO: Add junction resolution.
func ResolveLink(path string) (string, error) {
	return os.Readlink(path)
}

// SupportedLinkTypes returns link types available on Windows.
func SupportedLinkTypes() []string {
	return []string{"symlink", "junction"}
}
