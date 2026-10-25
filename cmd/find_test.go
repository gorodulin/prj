package cmd

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gorodulin/prj/internal/config"
)

// newFindTestConfig builds a fixture with two local projects and one
// metadata-only project (p20260103c has metadata but no folder).
func newFindTestConfig(t *testing.T) (config.Config, string) {
	t.Helper()
	cfg, tmp := newEditTestCfg(t)
	for _, id := range []string{"p20260101a", "p20260102b"} {
		mustMkProjectDir(t, cfg, id)
	}
	mustSeedSnapshot(t, cfg, "p20260103c", []string{"moto"})
	cfgPath := filepath.Join(tmp, "config.json")
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return cfg, cfgPath
}

// runFindCommand invokes the find command and captures stdout.
func runFindCommand(t *testing.T, args []string) (string, error) {
	t.Helper()
	stdout, _, err := runCommandCapture(t, findCmd, args)
	return stdout, err
}

func TestFind_RequiresQueryTerm(t *testing.T) {
	_, cfgPath := newFindTestConfig(t)
	_, err := runFindCommand(t, []string{"find", "--config", cfgPath})
	if err == nil {
		t.Fatal("expected error for find without query terms, got nil")
	}
	if !strings.Contains(err.Error(), "requires at least one query term or --tags") {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestFind_EmptyTagsRejected(t *testing.T) {
	_, cfgPath := newFindTestConfig(t)
	_, err := runFindCommand(t, []string{"find", "--config", cfgPath, "--tags", " , "})
	if err == nil {
		t.Fatal("expected error for find with blank --tags and no query terms, got nil")
	}
}

func TestFind_TagsOnly(t *testing.T) {
	_, cfgPath := newFindTestConfig(t)
	out, err := runFindCommand(t, []string{"find", "--config", cfgPath, "--no-color", "--tags", "moto"})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if !strings.Contains(out, "p20260103c") {
		t.Errorf("output missing tagged project p20260103c: %q", out)
	}
	if strings.Contains(out, "p20260101a") || strings.Contains(out, "p20260102b") {
		t.Errorf("--tags alone must filter to tagged projects, got: %q", out)
	}
}

func TestFind_IncludesMetadataOnly(t *testing.T) {
	_, cfgPath := newFindTestConfig(t)
	out, err := runFindCommand(t, []string{"find", "--config", cfgPath, "--no-color", "p2026"})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	for _, id := range []string{"p20260101a", "p20260102b", "p20260103c"} {
		if !strings.Contains(out, id) {
			t.Errorf("output missing %s (metadata-only projects must be searched): %q", id, out)
		}
	}
}

func TestFind_ORAcrossTerms(t *testing.T) {
	_, cfgPath := newFindTestConfig(t)
	out, err := runFindCommand(t, []string{"find", "--config", cfgPath, "--no-color", "0101", "0103"})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if !strings.Contains(out, "p20260101a") || !strings.Contains(out, "p20260103c") {
		t.Errorf("OR terms should match both projects, got: %q", out)
	}
	if strings.Contains(out, "p20260102b") {
		t.Errorf("p20260102b should not match, got: %q", out)
	}
}

func TestFind_TagsANDsWithQuery(t *testing.T) {
	_, cfgPath := newFindTestConfig(t)
	out, err := runFindCommand(t, []string{"find", "--config", cfgPath, "--no-color", "p2026", "--tags", "moto"})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if !strings.Contains(out, "p20260103c") {
		t.Errorf("output missing tagged project p20260103c: %q", out)
	}
	if strings.Contains(out, "p20260101a") || strings.Contains(out, "p20260102b") {
		t.Errorf("--tags must AND with the query, got: %q", out)
	}
}

func TestFind_NoMatchMessage(t *testing.T) {
	_, cfgPath := newFindTestConfig(t)
	out, err := runFindCommand(t, []string{"find", "--config", cfgPath, "--no-color", "zzz-no-such"})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if !strings.Contains(out, "No projects match the given query.") {
		t.Errorf("expected no-match message, got: %q", out)
	}
}

func TestFind_MutuallyExclusive_JSONJSONL(t *testing.T) {
	_, cfgPath := newFindTestConfig(t)
	_, err := runFindCommand(t, []string{"find", "--config", cfgPath, "x", "--json", "--jsonl"})
	if err == nil {
		t.Fatal("expected mutual-exclusion error, got nil")
	}
	if !strings.Contains(err.Error(), "none of the others can be") {
		t.Errorf("error %q is not cobra's mutual-exclusion error", err.Error())
	}
}

func TestFind_JSONAlias(t *testing.T) {
	_, cfgPath := newFindTestConfig(t)

	aliasOut, err := runFindCommand(t, []string{"find", "--config", cfgPath, "--no-color", "p2026", "--json"})
	if err != nil {
		t.Fatalf("--json: %v", err)
	}
	explicitOut, err := runFindCommand(t, []string{"find", "--config", cfgPath, "--no-color", "p2026", "--format", "json"})
	if err != nil {
		t.Fatalf("--format json: %v", err)
	}
	if aliasOut != explicitOut {
		t.Errorf("--json output differs from --format json:\n  alias:    %q\n  explicit: %q", aliasOut, explicitOut)
	}
}

func TestFind_JSONLAlias(t *testing.T) {
	_, cfgPath := newFindTestConfig(t)

	aliasOut, err := runFindCommand(t, []string{"find", "--config", cfgPath, "--no-color", "p2026", "--jsonl"})
	if err != nil {
		t.Fatalf("--jsonl: %v", err)
	}
	explicitOut, err := runFindCommand(t, []string{"find", "--config", cfgPath, "--no-color", "p2026", "--format", "jsonl"})
	if err != nil {
		t.Fatalf("--format jsonl: %v", err)
	}
	if aliasOut != explicitOut {
		t.Errorf("--jsonl output differs from --format jsonl:\n  alias:    %q\n  explicit: %q", aliasOut, explicitOut)
	}
}

func TestFind_ConfigListFormatFallback(t *testing.T) {
	cfg, cfgPath := newFindTestConfig(t)
	cfg.ListFormat = "{{.ID}}!"
	cfgPath2 := filepath.Join(filepath.Dir(cfgPath), "config-listformat.json")
	if err := config.Save(cfg, cfgPath2); err != nil {
		t.Fatalf("save config: %v", err)
	}

	out, err := runFindCommand(t, []string{"find", "--config", cfgPath2, "--no-color", "0101"})
	if err != nil {
		t.Fatalf("find with list_format config: %v", err)
	}
	if !strings.Contains(out, "p20260101a!") {
		t.Errorf("list_format config should apply when --format is absent, got: %q", out)
	}

	out, err = runFindCommand(t, []string{"find", "--config", cfgPath2, "--no-color", "0101", "-f", "{{.ID}}?"})
	if err != nil {
		t.Fatalf("find with explicit --format: %v", err)
	}
	if !strings.Contains(out, "p20260101a?") || strings.Contains(out, "!") {
		t.Errorf("explicit --format should override list_format config, got: %q", out)
	}
}

func TestFind_DashTermAfterDoubleDash(t *testing.T) {
	_, cfgPath := newFindTestConfig(t)
	out, err := runFindCommand(t, []string{"find", "--config", cfgPath, "--no-color", "--", "-foo"})
	if err != nil {
		t.Fatalf("find with -- -foo: %v", err)
	}
	if !strings.Contains(out, "No projects match the given query.") {
		t.Errorf("-foo after -- should be a query term (no match expected), got: %q", out)
	}
}

func TestFind_HideTags(t *testing.T) {
	cfgPath := newHideTagsTestConfig(t)

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"hidden by default", []string{"p2026"}, []string{"p20260101a", "p20260105a"}},
		{"--no-hide shows hidden", []string{"p2026", "--no-hide"}, []string{"p20260101a", "p20260102a", "p20260103a", "p20260104a", "p20260105a"}},
		{"--all shows hidden", []string{"p2026", "--all"}, []string{"p20260101a", "p20260102a", "p20260103a", "p20260104a", "p20260105a"}},
		{"--local narrows to local", []string{"p2026", "--local"}, []string{"p20260101a"}},
		{"--local --no-hide", []string{"p2026", "--local", "--no-hide"}, []string{"p20260101a", "p20260102a", "p20260103a"}},
		{"--missing narrows to metadata-only", []string{"p2026", "--missing"}, []string{"p20260105a"}},
		{"--tags archived lifts hiding", []string{"--tags", "archived"}, []string{"p20260102a", "p20260104a"}},
		{"--tags cli does not un-hide archived", []string{"--tags", "cli"}, []string{"p20260101a"}},
		{"query term does not lift hiding", []string{"archived"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"find", "--config", cfgPath, "--no-color", "-f", "{{.ID}}"}, tt.args...)
			stdout, err := runFindCommand(t, args)
			if err != nil {
				t.Fatalf("find %v: %v", tt.args, err)
			}
			if tt.want == nil {
				if !strings.Contains(stdout, "No projects match the given query.") {
					t.Errorf("expected no-match message, got %q", stdout)
				}
				return
			}
			if got := strings.Fields(stdout); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
