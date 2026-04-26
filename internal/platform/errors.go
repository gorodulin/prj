package platform

import "fmt"

// SymlinkPrivilegeError indicates that os.Symlink failed with
// ERROR_PRIVILEGE_NOT_HELD on Windows. It carries enough context for
// the cmd layer to format a recommendation tailored to the situation:
// FellBackFromJunction is true when junction was the configured kind
// but the link had to fall back to symlink (e.g. cross-volume).
type SymlinkPrivilegeError struct {
	LinkPath             string
	Target               string
	FellBackFromJunction bool
}

func (e *SymlinkPrivilegeError) Error() string {
	if e.FellBackFromJunction {
		return fmt.Sprintf("symlink %s -> %s requires Developer Mode or admin (junction not possible across volumes)", e.LinkPath, e.Target)
	}
	return fmt.Sprintf("symlink %s requires Developer Mode or admin", e.LinkPath)
}
