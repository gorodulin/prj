package linktree

import (
	"testing"
	"text/template"

	"github.com/gorodulin/prj/internal/format"
)

var testFM = format.FuncMap(false)

func testRender(t *testing.T, tmplStr string, entry ProjectEntry) string {
	t.Helper()
	tmpl, err := template.New("test").Funcs(testFM).Parse(tmplStr)
	if err != nil {
		t.Fatalf("parse template %q: %v", tmplStr, err)
	}
	return renderLinkName(tmpl, entry)
}

func TestRenderLinkName(t *testing.T) {
	tests := []struct {
		name   string
		format string
		id     string
		title  string
		want   string
	}{
		{"title with title", "{{.Title}}", "p20260301a", "Utils", "Utils"},
		{"title no title falls back to id", "{{.Title}}", "p20260301a", "", "p20260301a"},
		{"id format", "{{.ID}}", "p20260301a", "Utils", "p20260301a"},
		{"title and id", "{{.Title}} [{{.ID}}]", "p20260301a", "Utils", "Utils [p20260301a]"},
		{"empty format falls back to id", "", "p20260301a", "Utils", "p20260301a"},
		{"whitespace-only result falls back to id", "  {{.Title}}  ", "p20260301a", "", "p20260301a"},
		{"slash in title replaced", "{{.ID}} {{.Title}}", "p20260301a", "canoe/kayak route", "p20260301a canoe-kayak route"},
		{"backslash replaced", "{{.Title}}", "p20260301a", "foo\\bar", "foo-bar"},
		{"colon replaced", "{{.Title}}", "p20260301a", "HH:MM:SS", "HH-MM-SS"},
		{"angle brackets replaced", "{{.Title}}", "p20260301a", "<draft>", "-draft-"},
		{"question mark replaced", "{{.Title}}", "p20260301a", "What?", "What-"},
		{"asterisk replaced", "{{.Title}}", "p20260301a", "a*b", "a-b"},
		{"pipe in title replaced", "{{.Title}}", "p20260301a", "a|b", "a-b"},
		{"double quote replaced", "{{.Title}}", "p20260301a", `a "b" c`, "a -b- c"},
		{"control chars replaced", "{{.Title}}", "p20260301a", "a\x00b\x1Fc", "a-b-c"},
		{"trailing dots trimmed", "{{.Title}}", "p20260301a", "name...", "name"},
		{"trailing spaces trimmed", "{{.Title}}", "p20260301a", "name   ", "name"},
		{"windows reserved name mangled", "{{.Title}}", "p20260301a", "CON", "_CON"},
		{"windows reserved case insensitive", "{{.Title}}", "p20260301a", "nul", "_nul"},
		{"windows reserved with extension", "{{.Title}}", "p20260301a", "con.txt", "_con.txt"},
		{"non-reserved passes through", "{{.Title}}", "p20260301a", "CONNECT", "CONNECT"},
		{"all-unsafe title falls back to id", "{{.Title}}", "p20260301a", "???***", "p20260301a"},
		{"all-dashes title falls back to id", "{{.Title}}", "p20260301a", "///", "p20260301a"},
		{"upper function", "{{.Title | upper}}", "p20260301a", "Utils", "UTILS"},
		{"date function", "{{.ID | date \"YYYY\"}} {{.Title}}", "p20260301a", "Utils", "2026 Utils"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testRender(t, tt.format, ProjectEntry{ID: tt.id, Title: tt.title})
			if got != tt.want {
				t.Errorf("renderLinkName(%q, %q, %q) = %q, want %q", tt.format, tt.id, tt.title, got, tt.want)
			}
		})
	}
}

func TestTruncateUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		max      int
		wantLen  int
		wantLast byte // last byte should NOT be a continuation byte
	}{
		{"ascii fits", "hello", 10, 5, 'o'},
		{"ascii truncated", "hello world", 5, 5, 'o'},
		{"2-byte chars clean cut", "ааа", 4, 4, 0xb0},   // 2 full Cyrillic а (each 2 bytes)
		{"2-byte chars avoids split", "ааа", 3, 2, 0xb0}, // can't fit 1.5 chars, back to 1 char
		{"3-byte emoji avoids split", "a🎉b", 4, 1, 'a'}, // 🎉 is 4 bytes, won't fit after 'a'
		{"empty", "", 10, 0, 0},
		{"zero max", "hello", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateUTF8(tt.input, tt.max)
			if len(got) != tt.wantLen {
				t.Errorf("truncateUTF8(%q, %d) len = %d, want %d (got %q)", tt.input, tt.max, len(got), tt.wantLen, got)
			}
		})
	}
}

func TestMigrateOldFormat(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"old title", "{title}", "{{.Title}}"},
		{"old id", "{id}", "{{.ID}}"},
		{"old long tokens", "{project_id} {project_title}", "{{.ID}} {{.Title}}"},
		{"new syntax unchanged", "{{.Title}}", "{{.Title}}"},
		{"new syntax with pipe unchanged", "{{.ID | upper}}", "{{.ID | upper}}"},
		{"plain text unchanged", "hello", "hello"},
		{"empty unchanged", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := migrateOldFormat(tt.in)
			if got != tt.want {
				t.Errorf("migrateOldFormat(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolveNames(t *testing.T) {
	tests := []struct {
		name     string
		projects []ProjectEntry
		format   string
		want     map[string]string
	}{
		{
			name:     "unique titles no collision",
			projects: []ProjectEntry{{ID: "id1", Title: "Alpha"}, {ID: "id2", Title: "Beta"}},
			format:   "{{.Title}}",
			want:     map[string]string{"id1": "Alpha", "id2": "Beta"},
		},
		{
			name:     "collision both get id suffix",
			projects: []ProjectEntry{{ID: "id1", Title: "Utils"}, {ID: "id2", Title: "Utils"}},
			format:   "{{.Title}}",
			want:     map[string]string{"id1": "Utils (id1)", "id2": "Utils (id2)"},
		},
		{
			name: "three way collision",
			projects: []ProjectEntry{
				{ID: "id1", Title: "Utils"},
				{ID: "id2", Title: "Utils"},
				{ID: "id3", Title: "Utils"},
			},
			format: "{{.Title}}",
			want: map[string]string{
				"id1": "Utils (id1)",
				"id2": "Utils (id2)",
				"id3": "Utils (id3)",
			},
		},
		{
			name: "mixed collision and unique",
			projects: []ProjectEntry{
				{ID: "id1", Title: "Utils"},
				{ID: "id2", Title: "Utils"},
				{ID: "id3", Title: "Other"},
			},
			format: "{{.Title}}",
			want: map[string]string{
				"id1": "Utils (id1)",
				"id2": "Utils (id2)",
				"id3": "Other",
			},
		},
		{
			name:     "no title uses id which is unique",
			projects: []ProjectEntry{{ID: "id1", Title: ""}, {ID: "id2", Title: ""}},
			format:   "{{.Title}}",
			want:     map[string]string{"id1": "id1", "id2": "id2"},
		},
		{
			name:     "format with ID skips collision suffix",
			projects: []ProjectEntry{{ID: "id1", Title: "Utils"}, {ID: "id2", Title: "Utils"}},
			format:   "{{.ID}} {{.Title}}",
			want:     map[string]string{"id1": "id1 Utils", "id2": "id2 Utils"},
		},
		{
			name:     "format with piped ID skips collision suffix",
			projects: []ProjectEntry{{ID: "id1", Title: "Utils"}, {ID: "id2", Title: "Utils"}},
			format:   "{{.ID | upper}} {{.Title}}",
			want:     map[string]string{"id1": "ID1 Utils", "id2": "ID2 Utils"},
		},
		{
			name:     "single project no collision",
			projects: []ProjectEntry{{ID: "id1", Title: "Solo"}},
			format:   "{{.Title}}",
			want:     map[string]string{"id1": "Solo"},
		},
		{
			name:     "old format auto-migrated",
			projects: []ProjectEntry{{ID: "id1", Title: "Alpha"}, {ID: "id2", Title: "Beta"}},
			format:   "{title}",
			want:     map[string]string{"id1": "Alpha", "id2": "Beta"},
		},
		{
			name:     "old format with id auto-migrated and skips collision",
			projects: []ProjectEntry{{ID: "id1", Title: "Utils"}, {ID: "id2", Title: "Utils"}},
			format:   "{project_id} {project_title}",
			want:     map[string]string{"id1": "id1 Utils", "id2": "id2 Utils"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveNames(tt.projects, tt.format, testFM)
			if len(got) != len(tt.want) {
				t.Fatalf("ResolveNames returned %d entries, want %d", len(got), len(tt.want))
			}
			for id, wantName := range tt.want {
				if got[id] != wantName {
					t.Errorf("ResolveNames[%q] = %q, want %q", id, got[id], wantName)
				}
			}
		})
	}
}
