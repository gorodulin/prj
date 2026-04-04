package project

import (
	"fmt"
	"strings"
)

// BuildReadme generates README.md content with YAML front matter.
// Tags should already be normalized before calling this.
func BuildReadme(title string, tags []string) string {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("title: %s\n", title))
	if len(tags) > 0 {
		b.WriteString(fmt.Sprintf("tags: [%s]\n", strings.Join(tags, ", ")))
	}
	b.WriteString("---\n\n")
	b.WriteString(fmt.Sprintf("# %s\n", title))

	return b.String()
}
