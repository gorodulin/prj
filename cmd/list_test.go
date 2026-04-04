package cmd

import (
	"testing"

	"github.com/gorodulin/prj/internal/project"
)

func TestMatchesQuery(t *testing.T) {
	p := project.Project{
		ID:    "p20260402a",
		Title: "My Cool Project",
		Tags:  []string{"cli", "raspberry-pi", "golang"},
	}

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{"empty query matches all", "", true},
		{"matches ID", "p2026", true},
		{"matches title", "cool", true},
		{"matches title case-insensitive", "COOL", true},
		{"matches tag exactly", "cli", true},
		{"matches tag substring", "raspberry", true},
		{"matches tag case-insensitive", "GOLANG", true},
		{"no match", "python", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesQuery(p, tt.query)
			if got != tt.want {
				t.Errorf("matchesQuery(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestMatchesTags(t *testing.T) {
	p := project.Project{
		ID:   "p20260402a",
		Tags: []string{"cli", "golang", "raspberry-pi"},
	}

	tests := []struct {
		name string
		tags []string
		want bool
	}{
		{"empty tags matches all", nil, true},
		{"single tag present", []string{"cli"}, true},
		{"single tag absent", []string{"python"}, false},
		{"multiple tags all present", []string{"cli", "golang"}, true},
		{"multiple tags partial match", []string{"cli", "python"}, false},
		{"substring does not match", []string{"raspberry"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesTags(p, tt.tags)
			if got != tt.want {
				t.Errorf("matchesTags(%v) = %v, want %v", tt.tags, got, tt.want)
			}
		})
	}
}

func TestFilterProjects(t *testing.T) {
	projects := []project.Project{
		{ID: "p20260401a", Title: "Alpha CLI", Tags: []string{"cli", "golang"}},
		{ID: "p20260402a", Title: "Beta Server", Tags: []string{"server", "golang"}},
		{ID: "p20260403a", Title: "Gamma CLI", Tags: []string{"cli", "python"}},
	}

	tests := []struct {
		name    string
		query   string
		tags    []string
		wantIDs []string
	}{
		{"no filters", "", nil, []string{"p20260401a", "p20260402a", "p20260403a"}},
		{"query only", "cli", nil, []string{"p20260401a", "p20260403a"}},
		{"tag only", "", []string{"golang"}, []string{"p20260401a", "p20260402a"}},
		{"query and tag", "alpha", []string{"golang"}, []string{"p20260401a"}},
		{"no match", "nonexistent", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterProjects(projects, tt.query, tt.tags)
			gotIDs := make([]string, len(got))
			for i, p := range got {
				gotIDs[i] = p.ID
			}
			if len(gotIDs) != len(tt.wantIDs) {
				t.Fatalf("got %v, want %v", gotIDs, tt.wantIDs)
			}
			for i := range gotIDs {
				if gotIDs[i] != tt.wantIDs[i] {
					t.Errorf("got[%d] = %s, want %s", i, gotIDs[i], tt.wantIDs[i])
				}
			}
		})
	}
}
