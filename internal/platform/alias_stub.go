//go:build !(darwin && cgo)

package platform

import "fmt"

// CreateAlias is a stub that returns an error on platforms without cgo support.
func CreateAlias(aliasPath, targetPath string) error {
	return fmt.Errorf("create alias: finder aliases require macOS with cgo enabled")
}

// ResolveAlias is a stub that returns an error on platforms without cgo support.
func ResolveAlias(path string) (string, error) {
	return "", fmt.Errorf("resolve alias: finder aliases require macOS with cgo enabled")
}
