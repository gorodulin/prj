// Package platform provides OS-specific operations.
//
// Each platform_*.go file implements the same set of functions
// using build tags. The compiler includes exactly one per target OS.
//
// Functions provided by platform files:
//   - ResolveLink(path string) (target, kind string, err error)
//   - SupportedLinkTypes() []string
//   - DefaultLinkKind() string
//
// Functions provided by alias files (cgo on macOS, stub elsewhere):
//   - CreateAlias(aliasPath, targetPath string) error
//   - ResolveAlias(path string) (string, error)
package platform
