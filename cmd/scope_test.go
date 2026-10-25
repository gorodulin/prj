package cmd

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/project"
)

// newHideTagsTestConfig builds a fixture with hide_tags = archived,done:
//
//	p20260101a  local          cli            visible
//	p20260102a  local          archived, cli  hidden
//	p20260103a  local          done           hidden (newest local)
//	p20260104a  metadata-only  archived       hidden
//	p20260105a  metadata-only  moto           visible (newest overall)
func newHideTagsTestConfig(t *testing.T) string {
	t.Helper()
	cfg, tmp := newEditTestCfg(t)
	cfg.HideTags = []string{"archived", "done"}
	for _, id := range []string{"p20260101a", "p20260102a", "p20260103a"} {
		mustMkProjectDir(t, cfg, id)
	}
	mustSeedSnapshot(t, cfg, "p20260101a", []string{"cli"})
	mustSeedSnapshot(t, cfg, "p20260102a", []string{"archived", "cli"})
	mustSeedSnapshot(t, cfg, "p20260103a", []string{"done"})
	mustSeedSnapshot(t, cfg, "p20260104a", []string{"archived"})
	mustSeedSnapshot(t, cfg, "p20260105a", []string{"moto"})
	cfgPath := filepath.Join(tmp, "config.json")
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return cfgPath
}

func projectIDs(projects []project.Project) []string {
	ids := make([]string, 0, len(projects))
	for _, p := range projects {
		ids = append(ids, p.ID)
	}
	return ids
}

func TestResolveScope(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		defaultPresence presence
		want            listScope
	}{
		{"default local", nil, presenceLocal, listScope{presenceLocal, false}},
		{"default all", nil, presenceAll, listScope{presenceAll, false}},
		{"--local overrides default", []string{"--local"}, presenceAll, listScope{presenceLocal, false}},
		{"--missing", []string{"--missing"}, presenceLocal, listScope{presenceMissing, false}},
		{"--all implies --no-hide", []string{"--all"}, presenceLocal, listScope{presenceAll, true}},
		{"-a shorthand", []string{"-a"}, presenceLocal, listScope{presenceAll, true}},
		{"--no-hide keeps presence", []string{"--no-hide"}, presenceLocal, listScope{presenceLocal, true}},
		{"--local --no-hide compose", []string{"--local", "--no-hide"}, presenceAll, listScope{presenceLocal, true}},
		{"--missing --no-hide compose", []string{"--missing", "--no-hide"}, presenceLocal, listScope{presenceMissing, true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			addScopeFlags(cmd)
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("parse flags: %v", err)
			}
			if got := resolveScope(cmd, tt.defaultPresence); got != tt.want {
				t.Errorf("resolveScope(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestScopeFlagsMutuallyExclusive(t *testing.T) {
	tests := [][]string{
		{"--local", "--missing"},
		{"--local", "--all"},
		{"--missing", "--all"},
	}

	for _, args := range tests {
		t.Run(args[0]+" "+args[1], func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			addScopeFlags(cmd)
			if err := cmd.ParseFlags(args); err != nil {
				t.Fatalf("parse flags: %v", err)
			}
			if err := cmd.ValidateFlagGroups(); err == nil {
				t.Errorf("%v: expected mutual-exclusion error, got nil", args)
			}
		})
	}
}

func TestActiveHideTags(t *testing.T) {
	configured := []string{"archived"}
	tests := []struct {
		name  string
		scope listScope
		want  []string
	}{
		{"hiding applies", listScope{presenceLocal, false}, configured},
		{"hiding lifted", listScope{presenceLocal, true}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.scope.activeHideTags(configured); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("activeHideTags = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestByPresence(t *testing.T) {
	ids := idSets{
		all:      []string{"a", "b", "c"},
		localSet: map[string]bool{"a": true, "c": true},
	}
	tests := []struct {
		name     string
		presence presence
		want     []string
	}{
		{"local", presenceLocal, []string{"a", "c"}},
		{"missing", presenceMissing, []string{"b"}},
		{"all", presenceAll, []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ids.byPresence(tt.presence); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("byPresence = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsHiddenByTags(t *testing.T) {
	hideTags := []string{"archived", "done"}
	tests := []struct {
		name         string
		tags         []string
		hideTags     []string
		explicitTags []string
		want         bool
	}{
		{"no tags", nil, hideTags, nil, false},
		{"unrelated tags", []string{"cli"}, hideTags, nil, false},
		{"carries a hide tag", []string{"archived"}, hideTags, nil, true},
		{"hide tag among others", []string{"cli", "done"}, hideTags, nil, true},
		{"no hide tags configured", []string{"archived"}, nil, nil, false},
		{"explicit tag lifts its hide tag", []string{"archived"}, hideTags, []string{"archived"}, false},
		{"unrelated explicit tag does not lift", []string{"archived", "cli"}, hideTags, []string{"cli"}, true},
		{"explicit tag lifts only the tag it names", []string{"archived", "done"}, hideTags, []string{"archived"}, true},
		{"explicit tags lift every named hide tag", []string{"archived", "done"}, hideTags, []string{"archived", "done"}, false},
		{"substring is not a match", []string{"archived-2024"}, hideTags, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := project.Project{ID: "p20260101a", Tags: tt.tags}
			if got := isHiddenByTags(p, tt.hideTags, tt.explicitTags); got != tt.want {
				t.Errorf("isHiddenByTags(tags=%v, hide=%v, explicit=%v) = %v, want %v",
					tt.tags, tt.hideTags, tt.explicitTags, got, tt.want)
			}
		})
	}
}

func TestPartitionHidden(t *testing.T) {
	projects := []project.Project{
		{ID: "p20260101a", Tags: []string{"cli"}},
		{ID: "p20260102a", Tags: []string{"archived", "cli"}},
		{ID: "p20260103a", Tags: []string{"done"}},
		{ID: "p20260104a"},
	}

	tests := []struct {
		name         string
		hideTags     []string
		explicitTags []string
		wantIDs      []string
		wantHidden   int
	}{
		{"no hide tags", nil, nil, []string{"p20260101a", "p20260102a", "p20260103a", "p20260104a"}, 0},
		{"hides tagged", []string{"archived", "done"}, nil, []string{"p20260101a", "p20260104a"}, 2},
		{"explicit tag lifts", []string{"archived", "done"}, []string{"archived"}, []string{"p20260101a", "p20260102a", "p20260104a"}, 1},
		{"unrelated explicit tag keeps hiding", []string{"archived"}, []string{"cli"}, []string{"p20260101a", "p20260103a", "p20260104a"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visible, hiddenCount := partitionHidden(projects, tt.hideTags, tt.explicitTags)
			if got := projectIDs(visible); !reflect.DeepEqual(got, tt.wantIDs) {
				t.Errorf("visible = %v, want %v", got, tt.wantIDs)
			}
			if hiddenCount != tt.wantHidden {
				t.Errorf("hiddenCount = %d, want %d", hiddenCount, tt.wantHidden)
			}
		})
	}
}

func TestPrintHiddenHint_NonTTYIsSilent(t *testing.T) {
	for _, hiddenCount := range []int{0, 3} {
		var buf bytes.Buffer
		printHiddenHint(&buf, hiddenCount)
		if buf.Len() != 0 {
			t.Errorf("printHiddenHint(non-TTY, %d) wrote %q, want nothing", hiddenCount, buf.String())
		}
	}
}
