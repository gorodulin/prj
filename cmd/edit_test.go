package cmd

import "testing"

func TestAddToTags(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		add      []string
		want     []string
	}{
		{
			name:     "add new tags",
			existing: []string{"a", "b"},
			add:      []string{"c", "d"},
			want:     []string{"a", "b", "c", "d"},
		},
		{
			name:     "add overlapping tags",
			existing: []string{"a", "b"},
			add:      []string{"b", "c"},
			want:     []string{"a", "b", "c"},
		},
		{
			name:     "add to empty",
			existing: nil,
			add:      []string{"a", "b"},
			want:     []string{"a", "b"},
		},
		{
			name:     "add empty",
			existing: []string{"a", "b"},
			add:      nil,
			want:     []string{"a", "b"},
		},
		{
			name:     "add all duplicates",
			existing: []string{"a", "b"},
			add:      []string{"a", "b"},
			want:     []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addToTags(tt.existing, tt.add)
			if !sliceEqual(got, tt.want) {
				t.Errorf("addToTags(%v, %v) = %v, want %v", tt.existing, tt.add, got, tt.want)
			}
		})
	}
}

func TestRemoveFromTags(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		remove   []string
		want     []string
	}{
		{
			name:     "remove existing tags",
			existing: []string{"a", "b", "c"},
			remove:   []string{"b"},
			want:     []string{"a", "c"},
		},
		{
			name:     "remove absent tags",
			existing: []string{"a", "b"},
			remove:   []string{"x", "y"},
			want:     []string{"a", "b"},
		},
		{
			name:     "remove all tags",
			existing: []string{"a", "b"},
			remove:   []string{"a", "b"},
			want:     []string{},
		},
		{
			name:     "remove from empty",
			existing: nil,
			remove:   []string{"a"},
			want:     []string{},
		},
		{
			name:     "remove empty",
			existing: []string{"a", "b"},
			remove:   nil,
			want:     []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeFromTags(tt.existing, tt.remove)
			if !sliceEqual(got, tt.want) {
				t.Errorf("removeFromTags(%v, %v) = %v, want %v", tt.existing, tt.remove, got, tt.want)
			}
		})
	}
}
