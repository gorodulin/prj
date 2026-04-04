package format

import (
	"fmt"
	"io"
	"strings"

	"github.com/gorodulin/prj/internal/project"
)

// DefaultFormat is the built-in template used when no format is configured.
const DefaultFormat = `{{.ID | red}} {{.Local | flag}} {{.ID | date "YY-MM-DD" | cyan}} {{.Title | yellow}} {{.Tags | join ", " | blue}}`

// Options controls formatter behavior.
type Options struct {
	Color bool // resolved by caller via IsTTY
}

// Format writes projects to w in the requested format.
// Named formats: "json", "jsonl".
// If format contains "{{", it is treated as a Go text/template string.
// Empty string uses DefaultFormat.
func Format(w io.Writer, projects []project.Project, format string, opts Options) error {
	switch {
	case format == "":
		return formatTemplate(w, projects, DefaultFormat, opts)
	case format == "json":
		return formatJSON(w, projects, opts)
	case format == "jsonl":
		return formatJSONL(w, projects, opts)
	case strings.Contains(format, "{{"):
		return formatTemplate(w, projects, format, opts)
	default:
		return fmt.Errorf("unknown format %q (use json, jsonl, or a Go template)", format)
	}
}

// projectJSON returns a JSON-serializable representation of a project.
// Tags is always a slice (never nil) to serialize as [].
func projectJSON(p project.Project) interface{} {
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	return struct {
		ID    string   `json:"id"`
		Local bool     `json:"local"`
		Title string   `json:"title"`
		Path  string   `json:"path"`
		Tags  []string `json:"tags"`
	}{p.ID, p.Local, p.Title, p.Path, tags}
}
