//go:build !windows

package platform

import "fmt"

// CreateJunction is a stub on non-Windows platforms. NTFS junctions
// are a Windows-only filesystem feature.
func CreateJunction(linkPath, target string) error {
	return fmt.Errorf("create junction: junctions require Windows")
}
