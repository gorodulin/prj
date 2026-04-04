//go:build !(darwin && cgo)

package platform

// FinderComment pairs a link path with its desired comment text.
type FinderComment struct {
	Path    string
	Comment string
}

// SetFinderComment is a no-op on non-macOS platforms.
func SetFinderComment(path, comment string) error { return nil }

// SetFinderComments is a no-op on non-macOS platforms.
func SetFinderComments(items []FinderComment) error { return nil }

// GetFinderComment is a no-op on non-macOS platforms.
func GetFinderComment(path string) (string, error) { return "", nil }

// GetFinderCommentRaw is a no-op on non-macOS platforms.
func GetFinderCommentRaw(path string) ([]byte, error) { return nil, nil }

// FinderCommentChanged is a no-op on non-macOS platforms.
func FinderCommentChanged(path string, encoded []byte) (bool, error) { return false, nil }
