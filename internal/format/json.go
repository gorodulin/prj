package format

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/gorodulin/prj/internal/project"
)

func formatJSON(w io.Writer, projects []project.Project, opts Options) error {
	items := make([]interface{}, len(projects))
	for i, p := range projects {
		items[i] = projectJSON(p)
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

func formatJSONL(w io.Writer, projects []project.Project, opts Options) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, p := range projects {
		if err := enc.Encode(projectJSON(p)); err != nil {
			return fmt.Errorf("encode jsonl: %w", err)
		}
	}
	return nil
}
