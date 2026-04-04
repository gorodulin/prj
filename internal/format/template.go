package format

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/gorodulin/prj/internal/project"
)

func formatTemplate(w io.Writer, projects []project.Project, tmplStr string, opts Options) error {
	// Interpret common escape sequences so shell-passed strings like
	// '{{.ID}}\t{{.Title}}' produce actual tabs/newlines.
	tmplStr = strings.NewReplacer(`\t`, "\t", `\n`, "\n").Replace(tmplStr)

	tmpl, err := template.New("list").Funcs(FuncMap(opts.Color)).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	for _, p := range projects {
		if err := tmpl.Execute(w, p); err != nil {
			return fmt.Errorf("execute template for %s: %w", p.ID, err)
		}
		fmt.Fprintln(w)
	}
	return nil
}
