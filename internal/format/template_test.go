package format

import (
	"bytes"
	"testing"

	"github.com/gorodulin/prj/internal/project"
)

func TestFormatTemplate(t *testing.T) {
	projects := []project.Project{
		{ID: "p20260101a", Title: "Alpha", Path: "/tmp/a", Tags: []string{"go", "cli"}},
		{ID: "p20260102b", Title: "", Path: "/tmp/b", Tags: nil},
	}

	tests := []struct {
		name    string
		tmpl    string
		opts    Options
		want    string
		wantErr bool
	}{
		{
			name: "basic field",
			tmpl: "{{.ID}}",
			want: "p20260101a\np20260102b\n",
		},
		{
			name: "tab separated via Go template",
			tmpl: "{{.ID}}\t{{.Title}}",
			want: "p20260101a\tAlpha\np20260102b\t\n",
		},
		{
			name: "literal backslash-t interpreted as tab",
			tmpl: `{{.ID}}\t{{.Title}}`,
			want: "p20260101a\tAlpha\np20260102b\t\n",
		},
		{
			name: "join pipe",
			tmpl: "{{.Tags | join \",\"}}",
			want: "go,cli\n\n",
		},
		{
			name: "color enabled",
			tmpl: "{{.ID | green}}",
			opts: Options{Color: true},
			want: "\033[32mp20260101a\033[0m\n\033[32mp20260102b\033[0m\n",
		},
		{
			name: "color disabled",
			tmpl: "{{.ID | green}}",
			opts: Options{Color: false},
			want: "p20260101a\np20260102b\n",
		},
		{
			name: "conditional",
			tmpl: "{{if .Title}}{{.Title}}{{else}}untitled{{end}}",
			want: "Alpha\nuntitled\n",
		},
		{
			name: "path field",
			tmpl: "{{.Path}}",
			want: "/tmp/a\n/tmp/b\n",
		},
		{
			name:    "invalid template",
			tmpl:    "{{.BadSyntax",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := formatTemplate(&buf, projects, tt.tmpl, tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("formatTemplate: %v", err)
			}
			got := buf.String()
			if got != tt.want {
				t.Errorf("output:\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}
