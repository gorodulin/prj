package project

import (
	"os"
	"path/filepath"

	"github.com/gorodulin/prj/internal/text"
)

// Project holds info about a discovered project.
type Project struct {
	ID    string
	Local bool   // true if project folder exists on this machine
	Path  string // empty if project folder doesn't exist locally
	Title string
	Tags  []string
}

// ReadmeTitle extracts the title from README.md in the given directory.
// Returns empty string if the file is missing or has no title.
func ReadmeTitle(dir string) string {
	f, err := os.Open(filepath.Join(dir, "README.md"))
	if err != nil {
		return ""
	}
	defer f.Close()
	return text.ExtractMarkdownTitle(f)
}
