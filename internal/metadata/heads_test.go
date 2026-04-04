package metadata

import "testing"

func ptr(s string) *string { return &s }

func TestFindHeads(t *testing.T) {
	tests := []struct {
		name      string
		snapshots []Snapshot
		wantIDs   []string
	}{
		{
			name: "single root is head",
			snapshots: []Snapshot{
				{Filename: "t0.json", BasedOn: nil},
			},
			wantIDs: []string{"t0.json"},
		},
		{
			name: "linear chain, last is head",
			snapshots: []Snapshot{
				{Filename: "t0.json", BasedOn: nil},
				{Filename: "t1.json", BasedOn: []string{"t0.json"}},
				{Filename: "t2.json", BasedOn: []string{"t1.json"}},
			},
			wantIDs: []string{"t2.json"},
		},
		{
			name: "fork produces two heads",
			snapshots: []Snapshot{
				{Filename: "t0.json", BasedOn: nil},
				{Filename: "t1a.json", BasedOn: []string{"t0.json"}},
				{Filename: "t1b.json", BasedOn: []string{"t0.json"}},
			},
			wantIDs: []string{"t1a.json", "t1b.json"},
		},
		{
			name: "merge resolves to one head",
			snapshots: []Snapshot{
				{Filename: "t0.json", BasedOn: nil},
				{Filename: "t1a.json", BasedOn: []string{"t0.json"}},
				{Filename: "t1b.json", BasedOn: []string{"t0.json"}},
				{Filename: "t2.json", BasedOn: []string{"t1a.json", "t1b.json"}},
			},
			wantIDs: []string{"t2.json"},
		},
		{
			name:      "empty",
			snapshots: nil,
			wantIDs:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			heads := FindHeads(tt.snapshots)
			gotIDs := make([]string, len(heads))
			for i, h := range heads {
				gotIDs[i] = h.Filename
			}
			if !sliceEqual(gotIDs, tt.wantIDs) {
				t.Errorf("FindHeads = %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}

func TestLatestHead(t *testing.T) {
	tests := []struct {
		name      string
		snapshots []Snapshot
		wantTitle string
		wantTags  []string
	}{
		{
			name:      "empty returns zero",
			snapshots: nil,
			wantTitle: "",
			wantTags:  nil,
		},
		{
			name: "single snapshot",
			snapshots: []Snapshot{
				{Filename: "t0.json", TitleSet: ptr("Hello"), Tags: []string{"a"}},
			},
			wantTitle: "Hello",
			wantTags:  []string{"a"},
		},
		{
			name: "linear chain picks last",
			snapshots: []Snapshot{
				{Filename: "t0.json", BasedOn: nil, TitleSet: ptr("Old")},
				{Filename: "t1.json", BasedOn: []string{"t0.json"}, TitleSet: ptr("New"), Tags: []string{"x"}},
			},
			wantTitle: "New",
			wantTags:  []string{"x"},
		},
		{
			name: "fork picks latest head by filename",
			snapshots: []Snapshot{
				{Filename: "20250101T000000Z.json", BasedOn: nil, TitleSet: ptr("Root")},
				{Filename: "20250102T000000Z.json", BasedOn: []string{"20250101T000000Z.json"}, TitleSet: ptr("Branch A"), Tags: []string{"a"}},
				{Filename: "20250103T000000Z.json", BasedOn: []string{"20250101T000000Z.json"}, TitleSet: ptr("Branch B"), Tags: []string{"b"}},
			},
			wantTitle: "Branch B",
			wantTags:  []string{"b"},
		},
		{
			name: "null title_set returns empty title",
			snapshots: []Snapshot{
				{Filename: "t0.json", TitleSet: nil, Tags: []string{"a"}},
			},
			wantTitle: "",
			wantTags:  []string{"a"},
		},
		{
			name: "null title_set inherits from earlier snapshot",
			snapshots: []Snapshot{
				{Filename: "t0.json", BasedOn: nil, TitleSet: ptr("Original")},
				{Filename: "t1.json", BasedOn: []string{"t0.json"}, TitleSet: nil, Tags: []string{"x"}},
			},
			wantTitle: "Original",
			wantTags:  []string{"x"},
		},
		{
			name: "all nil title_set returns empty",
			snapshots: []Snapshot{
				{Filename: "t0.json", BasedOn: nil, TitleSet: nil, Tags: []string{"a"}},
				{Filename: "t1.json", BasedOn: []string{"t0.json"}, TitleSet: nil, Tags: []string{"b"}},
			},
			wantTitle: "",
			wantTags:  []string{"b"},
		},
		{
			name: "cleared title inherits empty string not earlier value",
			snapshots: []Snapshot{
				{Filename: "t0.json", BasedOn: nil, TitleSet: ptr("Original")},
				{Filename: "t1.json", BasedOn: []string{"t0.json"}, TitleSet: ptr("")},
				{Filename: "t2.json", BasedOn: []string{"t1.json"}, TitleSet: nil, Tags: []string{"x"}},
			},
			wantTitle: "",
			wantTags:  []string{"x"},
		},
		{
			name: "fork inherits title from latest snapshot with title_set",
			snapshots: []Snapshot{
				{Filename: "t0.json", BasedOn: nil, TitleSet: ptr("Root")},
				{Filename: "t1a.json", BasedOn: []string{"t0.json"}, TitleSet: ptr("Branch A")},
				{Filename: "t1b.json", BasedOn: []string{"t0.json"}, TitleSet: nil, Tags: []string{"b"}},
			},
			wantTitle: "Branch A",
			wantTags:  []string{"b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := LatestHead(tt.snapshots)
			if meta.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", meta.Title, tt.wantTitle)
			}
			if !sliceEqual(meta.Tags, tt.wantTags) {
				t.Errorf("Tags = %v, want %v", meta.Tags, tt.wantTags)
			}
		})
	}
}

func TestTagDeltas(t *testing.T) {
	tests := []struct {
		name        string
		oldTags     []string
		newTags     []string
		wantAdded   []string
		wantRemoved []string
	}{
		{
			name:        "no change",
			oldTags:     []string{"a", "b"},
			newTags:     []string{"a", "b"},
			wantAdded:   nil,
			wantRemoved: nil,
		},
		{
			name:        "add only",
			oldTags:     []string{"a"},
			newTags:     []string{"a", "b", "c"},
			wantAdded:   []string{"b", "c"},
			wantRemoved: nil,
		},
		{
			name:        "remove only",
			oldTags:     []string{"a", "b", "c"},
			newTags:     []string{"a"},
			wantAdded:   nil,
			wantRemoved: []string{"b", "c"},
		},
		{
			name:        "add and remove",
			oldTags:     []string{"a", "b"},
			newTags:     []string{"b", "c"},
			wantAdded:   []string{"c"},
			wantRemoved: []string{"a"},
		},
		{
			name:        "both empty",
			oldTags:     nil,
			newTags:     nil,
			wantAdded:   nil,
			wantRemoved: nil,
		},
		{
			name:        "from empty",
			oldTags:     nil,
			newTags:     []string{"a", "b"},
			wantAdded:   []string{"a", "b"},
			wantRemoved: nil,
		},
		{
			name:        "to empty",
			oldTags:     []string{"a", "b"},
			newTags:     nil,
			wantAdded:   nil,
			wantRemoved: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			added, removed := TagDeltas(tt.oldTags, tt.newTags)
			if !sliceEqual(added, tt.wantAdded) {
				t.Errorf("added = %v, want %v", added, tt.wantAdded)
			}
			if !sliceEqual(removed, tt.wantRemoved) {
				t.Errorf("removed = %v, want %v", removed, tt.wantRemoved)
			}
		})
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
