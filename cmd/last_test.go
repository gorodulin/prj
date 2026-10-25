package cmd

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/project"
)

// newLastTestConfig builds a fixture with three local projects, one of
// them past suffix "z" to exercise creation order, plus the metadata-only
// project p20260103a, the newest overall.
func newLastTestConfig(t *testing.T) string {
	t.Helper()
	cfg, tmp := newEditTestCfg(t)
	for _, id := range []string{"p20260101a", "p20260102z", "p20260102aa"} {
		mustMkProjectDir(t, cfg, id)
	}
	mustSeedSnapshot(t, cfg, "p20260103a", []string{"moto"})
	cfgPath := filepath.Join(tmp, "config.json")
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return cfgPath
}

// runLastIDs runs "prj last" with an ID-only template and returns the IDs.
func runLastIDs(t *testing.T, args ...string) []string {
	t.Helper()
	cfgPath := newLastTestConfig(t)
	full := append([]string{"last", "--config", cfgPath, "--no-color", "-f", "{{.ID}}"}, args...)
	stdout, _, err := runCommandCapture(t, lastCmd, full)
	if err != nil {
		t.Fatalf("last %v: %v", args, err)
	}
	return strings.Fields(stdout)
}

func TestLast_DefaultsToOne(t *testing.T) {
	got := runLastIDs(t)
	if want := []string{"p20260102aa"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLast_N(t *testing.T) {
	got := runLastIDs(t, "2")
	if want := []string{"p20260102z", "p20260102aa"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v (oldest first, z before aa)", got, want)
	}
}

func TestLast_NExceedsCount(t *testing.T) {
	got := runLastIDs(t, "10")
	if want := []string{"p20260101a", "p20260102z", "p20260102aa"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLast_AllIncludesMetadataOnly(t *testing.T) {
	got := runLastIDs(t, "2", "--all")
	if want := []string{"p20260102aa", "p20260103a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLast_InvalidN(t *testing.T) {
	cfgPath := newLastTestConfig(t)
	for _, arg := range []string{"0", "-1", "five"} {
		_, _, err := runCommandCapture(t, lastCmd, []string{"last", "--config", cfgPath, "--", arg})
		if err == nil || !strings.Contains(err.Error(), "N must be a positive integer") {
			t.Errorf("last %s: got error %v, want positive-integer error", arg, err)
		}
	}
}

func TestLast_JSONAlias(t *testing.T) {
	cfgPath := newLastTestConfig(t)
	stdout, _, err := runCommandCapture(t, lastCmd, []string{"last", "--config", cfgPath, "2", "--json"})
	if err != nil {
		t.Fatalf("last --json: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2", len(items))
	}
}

func TestParseLastCount(t *testing.T) {
	tests := []struct {
		args    []string
		want    int
		wantErr bool
	}{
		{nil, 1, false},
		{[]string{"3"}, 3, false},
		{[]string{"0"}, 0, true},
		{[]string{"-2"}, 0, true},
		{[]string{"x"}, 0, true},
	}
	for _, tt := range tests {
		got, err := parseLastCount(tt.args)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("parseLastCount(%v) = %d, %v; want %d, err=%v", tt.args, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestLastN(t *testing.T) {
	ids := []string{"a", "b", "c"}
	tests := []struct {
		n    int
		want []string
	}{
		{1, []string{"c"}},
		{2, []string{"b", "c"}},
		{3, ids},
		{5, ids},
	}
	for _, tt := range tests {
		if got := lastN(ids, tt.n); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("lastN(%d) = %v, want %v", tt.n, got, tt.want)
		}
	}
}

func TestLast_HideTags(t *testing.T) {
	cfgPath := newHideTagsTestConfig(t)

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"skips hidden newest", nil, []string{"p20260101a"}},
		{"--no-hide returns hidden newest", []string{"--no-hide"}, []string{"p20260103a"}},
		{"N exceeds visible count", []string{"5"}, []string{"p20260101a"}},
		{"--no-hide N", []string{"2", "--no-hide"}, []string{"p20260102a", "p20260103a"}},
		{"--all includes hidden and metadata-only", []string{"2", "--all"}, []string{"p20260104a", "p20260105a"}},
		{"--missing still hides", []string{"2", "--missing"}, []string{"p20260105a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"last", "--config", cfgPath, "--no-color", "-f", "{{.ID}}"}, tt.args...)
			stdout, _, err := runCommandCapture(t, lastCmd, args)
			if err != nil {
				t.Fatalf("last %v: %v", tt.args, err)
			}
			if got := strings.Fields(stdout); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLastNVisible(t *testing.T) {
	tagsByID := map[string][]string{
		"a": {"cli"},
		"b": {"archived"},
		"c": nil,
		"d": {"done"},
		"e": {"archived", "cli"},
	}
	ids := []string{"a", "b", "c", "d", "e"}
	hideTags := []string{"archived", "done"}

	tests := []struct {
		name         string
		n            int
		hideTags     []string
		want         []string
		wantHidden   int
		wantResolved int
	}{
		{"one skips hidden newest", 1, hideTags, []string{"c"}, 2, 3},
		{"two", 2, hideTags, []string{"a", "c"}, 3, 5},
		{"N exceeds visible", 10, hideTags, []string{"a", "c"}, 3, 5},
		{"no hide tags", 2, nil, []string{"d", "e"}, 0, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := 0
			resolve := func(id string) project.Project {
				resolved++
				return project.Project{ID: id, Tags: tagsByID[id]}
			}
			got, hiddenCount := lastNVisible(ids, tt.n, resolve, tt.hideTags)
			if gotIDs := projectIDs(got); !reflect.DeepEqual(gotIDs, tt.want) {
				t.Errorf("got %v, want %v", gotIDs, tt.want)
			}
			if hiddenCount != tt.wantHidden {
				t.Errorf("hiddenCount = %d, want %d", hiddenCount, tt.wantHidden)
			}
			if resolved != tt.wantResolved {
				t.Errorf("resolved %d projects, want %d (walk must stop once N are collected)", resolved, tt.wantResolved)
			}
		})
	}
}
