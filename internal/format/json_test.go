package format

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/gorodulin/prj/internal/project"
)

func TestFormatJSON(t *testing.T) {
	tests := []struct {
		name     string
		projects []project.Project
		opts     Options
		check    func(t *testing.T, output string)
	}{
		{
			name: "all fields present",
			projects: []project.Project{
				{ID: "p20260101a", Title: "Alpha", Path: "/tmp/alpha", Tags: []string{"go", "cli"}},
			},
			check: func(t *testing.T, output string) {
				var items []map[string]interface{}
				if err := json.Unmarshal([]byte(output), &items); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if len(items) != 1 {
					t.Fatalf("got %d items, want 1", len(items))
				}
				item := items[0]
				if item["id"] != "p20260101a" {
					t.Errorf("id = %v", item["id"])
				}
				if item["title"] != "Alpha" {
					t.Errorf("title = %v", item["title"])
				}
				if item["path"] != "/tmp/alpha" {
					t.Errorf("path = %v", item["path"])
				}
				tags, ok := item["tags"].([]interface{})
				if !ok {
					t.Fatalf("tags is not an array: %T", item["tags"])
				}
				if len(tags) != 2 {
					t.Errorf("got %d tags, want 2", len(tags))
				}
			},
		},
		{
			name:     "empty list",
			projects: nil,
			check: func(t *testing.T, output string) {
				var items []interface{}
				if err := json.Unmarshal([]byte(output), &items); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if len(items) != 0 {
					t.Errorf("expected empty array, got %d items", len(items))
				}
			},
		},
		{
			name: "nil tags serializes as empty array",
			projects: []project.Project{
				{ID: "p20260101a", Tags: nil},
			},
			check: func(t *testing.T, output string) {
				var items []map[string]interface{}
				if err := json.Unmarshal([]byte(output), &items); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				tags, ok := items[0]["tags"].([]interface{})
				if !ok {
					t.Fatalf("tags should be array, got %T", items[0]["tags"])
				}
				if len(tags) != 0 {
					t.Errorf("nil tags should serialize as [], got %v", tags)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := formatJSON(&buf, tt.projects, tt.opts)
			if err != nil {
				t.Fatalf("formatJSON: %v", err)
			}
			tt.check(t, buf.String())
		})
	}
}

func TestFormatJSONL(t *testing.T) {
	tests := []struct {
		name     string
		projects []project.Project
		opts     Options
		check    func(t *testing.T, output string)
	}{
		{
			name: "one object per line",
			projects: []project.Project{
				{ID: "p20260101a", Title: "Alpha", Tags: []string{"go"}},
				{ID: "p20260102b", Title: "Beta", Tags: nil},
			},
			check: func(t *testing.T, output string) {
				lines := splitNonEmpty(output)
				if len(lines) != 2 {
					t.Fatalf("expected 2 lines, got %d: %q", len(lines), output)
				}
				for i, line := range lines {
					var m map[string]interface{}
					if err := json.Unmarshal([]byte(line), &m); err != nil {
						t.Errorf("line %d not valid JSON: %v", i, err)
					}
				}
			},
		},
		{
			name:     "empty list produces no output",
			projects: nil,
			check: func(t *testing.T, output string) {
				if output != "" {
					t.Errorf("expected empty output, got %q", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := formatJSONL(&buf, tt.projects, tt.opts)
			if err != nil {
				t.Fatalf("formatJSONL: %v", err)
			}
			tt.check(t, buf.String())
		})
	}
}

// splitNonEmpty splits s by newlines, dropping empty trailing entries.
func splitNonEmpty(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if i > start {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
