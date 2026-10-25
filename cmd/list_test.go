package cmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/project"
)

func TestMatchesAnyQueryTerm(t *testing.T) {
	p := project.Project{
		ID:    "p20260402a",
		Title: "My Cool Project",
		Tags:  []string{"cli", "raspberry-pi", "golang"},
	}

	tests := []struct {
		name  string
		terms []string
		want  bool
	}{
		{"empty terms match all", nil, true},
		{"matches ID", []string{"p2026"}, true},
		{"matches title", []string{"cool"}, true},
		{"matches tag exactly", []string{"cli"}, true},
		{"matches tag substring", []string{"raspberry"}, true},
		{"no match", []string{"python"}, false},
		{"any term matches (OR)", []string{"python", "cool"}, true},
		{"all terms miss", []string{"python", "rust"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesAnyQueryTerm(p, tt.terms)
			if got != tt.want {
				t.Errorf("matchesAnyQueryTerm(%v) = %v, want %v", tt.terms, got, tt.want)
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

func newListTestConfig(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	cfg := config.Config{
		ProjectsFolder:  filepath.Join(tmp, "Projects"),
		MetadataFolder:  filepath.Join(tmp, "Metadata"),
		MetadataSuffix:  ".meta",
		MachineID:       "test",
		MachineName:     "test",
		RetentionDays:   30,
		ProjectIDType:   project.FormatAYMDb,
		ProjectIDPrefix: "p",
	}
	if err := os.MkdirAll(cfg.ProjectsFolder, 0o755); err != nil {
		t.Fatalf("mkdir Projects: %v", err)
	}
	if err := os.MkdirAll(cfg.MetadataFolder, 0o755); err != nil {
		t.Fatalf("mkdir Metadata: %v", err)
	}
	for _, id := range []string{"p20260101a", "p20260102b"} {
		if err := os.MkdirAll(filepath.Join(cfg.ProjectsFolder, id), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", id, err)
		}
	}
	cfgPath := filepath.Join(tmp, "config.json")
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return cfgPath
}

// runCommandCapture invokes rootCmd with the given args, captures stdout and
// stderr, and resets globals plus the given command's flag state so
// successive calls don't bleed into each other.
func runCommandCapture(t *testing.T, cmd *cobra.Command, args []string) (stdout, stderr string, err error) {
	t.Helper()
	resetFlags := func(fs *pflag.FlagSet) {
		fs.VisitAll(func(f *pflag.Flag) {
			f.Changed = false
			_ = f.Value.Set(f.DefValue)
		})
	}
	resetFlags(rootCmd.PersistentFlags())
	resetFlags(cmd.Flags())
	cfgFile = ""
	noColor = false
	colorMode = ""

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = wOut, wErr

	rootCmd.SetArgs(args)
	execErr := rootCmd.Execute()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = oldOut, oldErr

	outBuf, _ := io.ReadAll(rOut)
	errBuf, _ := io.ReadAll(rErr)
	return string(outBuf), string(errBuf), execErr
}

// runListCommand invokes the list command and captures stdout.
func runListCommand(t *testing.T, args []string) (string, error) {
	t.Helper()
	stdout, _, err := runCommandCapture(t, listCmd, args)
	return stdout, err
}

func TestList_JSONAlias(t *testing.T) {
	cfgPath := newListTestConfig(t)

	aliasOut, err := runListCommand(t, []string{"list", "--config", cfgPath, "--no-color", "--json"})
	if err != nil {
		t.Fatalf("--json: %v", err)
	}
	explicitOut, err := runListCommand(t, []string{"list", "--config", cfgPath, "--no-color", "--format", "json"})
	if err != nil {
		t.Fatalf("--format json: %v", err)
	}
	if aliasOut != explicitOut {
		t.Errorf("--json output differs from --format json:\n  alias:    %q\n  explicit: %q", aliasOut, explicitOut)
	}
}

func TestList_JSONLAlias(t *testing.T) {
	cfgPath := newListTestConfig(t)

	aliasOut, err := runListCommand(t, []string{"list", "--config", cfgPath, "--no-color", "--jsonl"})
	if err != nil {
		t.Fatalf("--jsonl: %v", err)
	}
	explicitOut, err := runListCommand(t, []string{"list", "--config", cfgPath, "--no-color", "--format", "jsonl"})
	if err != nil {
		t.Fatalf("--format jsonl: %v", err)
	}
	if aliasOut != explicitOut {
		t.Errorf("--jsonl output differs from --format jsonl:\n  alias:    %q\n  explicit: %q", aliasOut, explicitOut)
	}
}

func TestList_MutuallyExclusive_JSONFormat(t *testing.T) {
	cfgPath := newListTestConfig(t)
	_, err := runListCommand(t, []string{"list", "--config", cfgPath, "--json", "--format", "jsonl"})
	if err == nil {
		t.Fatal("expected mutual-exclusion error, got nil")
	}
	if !strings.Contains(err.Error(), "none of the others can be") {
		t.Errorf("error %q is not cobra's mutual-exclusion error", err.Error())
	}
}

func TestList_MutuallyExclusive_JSONJSONL(t *testing.T) {
	cfgPath := newListTestConfig(t)
	_, err := runListCommand(t, []string{"list", "--config", cfgPath, "--json", "--jsonl"})
	if err == nil {
		t.Fatal("expected mutual-exclusion error, got nil")
	}
	if !strings.Contains(err.Error(), "none of the others can be") {
		t.Errorf("error %q is not cobra's mutual-exclusion error", err.Error())
	}
}

func TestList_ColorAlwaysEmitsANSIOnPipe(t *testing.T) {
	cfgPath := newListTestConfig(t)

	out, err := runListCommand(t, []string{"list", "--config", cfgPath, "--color", "always"})
	if err != nil {
		t.Fatalf("--color always: %v", err)
	}
	if !strings.Contains(out, "\033[") {
		t.Errorf("--color always on a pipe should emit ANSI escapes, got %q", out)
	}
}

func TestList_ColorInvalid(t *testing.T) {
	cfgPath := newListTestConfig(t)

	for _, tt := range []struct{ name, value string }{
		{"bogus", "bogus"},
		{"explicit empty", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runListCommand(t, []string{"list", "--config", cfgPath, "--color=" + tt.value})
			if err == nil {
				t.Fatalf("--color=%q: expected error, got nil", tt.value)
			}
			if !strings.Contains(err.Error(), "is not recognized") {
				t.Errorf("--color=%q: error %q is not the not-recognized error", tt.value, err.Error())
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
		terms   []string
		tags    []string
		wantIDs []string
	}{
		{"no filters", nil, nil, []string{"p20260401a", "p20260402a", "p20260403a"}},
		{"single term", []string{"cli"}, nil, []string{"p20260401a", "p20260403a"}},
		{"term case-insensitive", []string{"ALPHA"}, nil, []string{"p20260401a"}},
		{"multiple terms OR", []string{"alpha", "beta"}, nil, []string{"p20260401a", "p20260402a"}},
		{"tag only", nil, []string{"golang"}, []string{"p20260401a", "p20260402a"}},
		{"term and tag", []string{"alpha"}, []string{"golang"}, []string{"p20260401a"}},
		{"no match", []string{"nonexistent"}, nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterProjects(projects, tt.terms, tt.tags)
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

func TestList_PositionalTermFilters(t *testing.T) {
	cfgPath := newListTestConfig(t)

	out, err := runListCommand(t, []string{"list", "--config", cfgPath, "--no-color", "0101"})
	if err != nil {
		t.Fatalf("list with positional term: %v", err)
	}
	if !strings.Contains(out, "p20260101a") {
		t.Errorf("output missing p20260101a: %q", out)
	}
	if strings.Contains(out, "p20260102b") {
		t.Errorf("output should not contain p20260102b: %q", out)
	}
}

func TestList_PositionalTermsOR(t *testing.T) {
	cfgPath := newListTestConfig(t)

	out, err := runListCommand(t, []string{"list", "--config", cfgPath, "--no-color", "0101", "0102"})
	if err != nil {
		t.Fatalf("list with two terms: %v", err)
	}
	if !strings.Contains(out, "p20260101a") || !strings.Contains(out, "p20260102b") {
		t.Errorf("OR terms should match both projects, got: %q", out)
	}
}

func TestList_QueryFlagMergedWithPositional(t *testing.T) {
	cfgPath := newListTestConfig(t)

	out, err := runListCommand(t, []string{"list", "--config", cfgPath, "--no-color", "0102", "-q", "0101"})
	if err != nil {
		t.Fatalf("list with -q and positional term: %v", err)
	}
	if !strings.Contains(out, "p20260101a") || !strings.Contains(out, "p20260102b") {
		t.Errorf("-q should OR-merge with positional terms, got: %q", out)
	}
}

func TestList_NoMatchMessage(t *testing.T) {
	cfgPath := newListTestConfig(t)

	out, err := runListCommand(t, []string{"list", "--config", cfgPath, "--no-color", "zzz-no-such"})
	if err != nil {
		t.Fatalf("list with unmatched term: %v", err)
	}
	if !strings.Contains(out, "No projects match the given filter.") {
		t.Errorf("expected no-match message, got: %q", out)
	}
}

func TestList_DashTermAfterDoubleDash(t *testing.T) {
	cfgPath := newListTestConfig(t)

	out, err := runListCommand(t, []string{"list", "--config", cfgPath, "--no-color", "--", "-foo"})
	if err != nil {
		t.Fatalf("list with -- -foo: %v", err)
	}
	if !strings.Contains(out, "No projects match the given filter.") {
		t.Errorf("-foo after -- should be a query term (no match expected), got: %q", out)
	}
}

// TestList_CyrillicTermMatchesTag exercises the full CLI path with metadata:
// a non-ASCII term must match a tag case-insensitively (bilingual tags are
// the motivating use case for term queries).
func TestList_CyrillicTermMatchesTag(t *testing.T) {
	cfg, tmp := newEditTestCfg(t)
	mustMkProjectDir(t, cfg, "p20260101a")
	mustMkProjectDir(t, cfg, "p20260102b")
	mustSeedSnapshot(t, cfg, "p20260101a", []string{"мотоцикл"})
	cfgPath := filepath.Join(tmp, "config.json")
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	out, err := runListCommand(t, []string{"list", "--config", cfgPath, "--no-color", "МОТОЦИКЛ"})
	if err != nil {
		t.Fatalf("list with cyrillic term: %v", err)
	}
	if !strings.Contains(out, "p20260101a") {
		t.Errorf("uppercase cyrillic term should match lowercase tag, got: %q", out)
	}
	if strings.Contains(out, "p20260102b") {
		t.Errorf("untagged project should not match, got: %q", out)
	}
}

func TestList_ConfigListFormatFallback(t *testing.T) {
	cfg, tmp := newEditTestCfg(t)
	mustMkProjectDir(t, cfg, "p20260101a")
	cfg.ListFormat = "{{.ID}}!"
	cfgPath := filepath.Join(tmp, "config.json")
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	out, err := runListCommand(t, []string{"list", "--config", cfgPath, "--no-color"})
	if err != nil {
		t.Fatalf("list with list_format config: %v", err)
	}
	if !strings.Contains(out, "p20260101a!") {
		t.Errorf("list_format config should apply when --format is absent, got: %q", out)
	}

	out, err = runListCommand(t, []string{"list", "--config", cfgPath, "--no-color", "-f", "{{.ID}}?"})
	if err != nil {
		t.Fatalf("list with explicit --format: %v", err)
	}
	if !strings.Contains(out, "p20260101a?") || strings.Contains(out, "!") {
		t.Errorf("explicit --format should override list_format config, got: %q", out)
	}
}

func TestList_TagAlias(t *testing.T) {
	cfgPath := newListTestConfig(t)

	aliasOut, err := runListCommand(t, []string{"list", "--config", cfgPath, "--no-color", "--tag", "wip,cli"})
	if err != nil {
		t.Fatalf("--tag: %v", err)
	}
	explicitOut, err := runListCommand(t, []string{"list", "--config", cfgPath, "--no-color", "--tags", "wip,cli"})
	if err != nil {
		t.Fatalf("--tags: %v", err)
	}
	if aliasOut != explicitOut {
		t.Errorf("--tag output differs from --tags:\n  alias:    %q\n  explicit: %q", aliasOut, explicitOut)
	}
}

func TestList_TagsTagMutuallyExclusive(t *testing.T) {
	cfgPath := newListTestConfig(t)
	_, err := runListCommand(t, []string{"list", "--config", cfgPath, "--tags", "x", "--tag", "y"})
	if err == nil {
		t.Fatal("expected mutual-exclusion error, got nil")
	}
	if !strings.Contains(err.Error(), "none of the others can be") {
		t.Errorf("error %q is not cobra's mutual-exclusion error", err.Error())
	}
}

func TestList_HideTags(t *testing.T) {
	cfgPath := newHideTagsTestConfig(t)

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"hidden by default", nil, []string{"p20260101a"}},
		{"--local equals default", []string{"--local"}, []string{"p20260101a"}},
		{"--no-hide shows hidden local", []string{"--no-hide"}, []string{"p20260101a", "p20260102a", "p20260103a"}},
		{"--all shows everything", []string{"--all"}, []string{"p20260101a", "p20260102a", "p20260103a", "p20260104a", "p20260105a"}},
		{"--missing still hides", []string{"--missing"}, []string{"p20260105a"}},
		{"--missing --no-hide", []string{"--missing", "--no-hide"}, []string{"p20260104a", "p20260105a"}},
		{"--tags archived lifts hiding", []string{"--tags", "archived"}, []string{"p20260102a"}},
		{"--tags cli does not un-hide archived", []string{"--tags", "cli"}, []string{"p20260101a"}},
		{"query term does not lift hiding", []string{"0102"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"list", "--config", cfgPath, "--no-color", "-f", "{{.ID}}"}, tt.args...)
			stdout, _, err := runCommandCapture(t, listCmd, args)
			if err != nil {
				t.Fatalf("list %v: %v", tt.args, err)
			}
			got := strings.Fields(stdout)
			if tt.want == nil {
				if !strings.Contains(stdout, "No projects match the given filter.") {
					t.Errorf("expected no-match message, got %q", stdout)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestList_ScopeFlagsMutuallyExclusive(t *testing.T) {
	cfgPath := newHideTagsTestConfig(t)
	for _, pair := range [][]string{{"--local", "--missing"}, {"--local", "--all"}, {"--missing", "--all"}} {
		args := append([]string{"list", "--config", cfgPath}, pair...)
		_, _, err := runCommandCapture(t, listCmd, args)
		if err == nil || !strings.Contains(err.Error(), "none of the others can be") {
			t.Errorf("list %v: got error %v, want mutual-exclusion error", pair, err)
		}
	}
}

// TestList_HideTagsJSONStdoutClean checks that hiding never leaks into
// stdout: the JSON parses and contains only visible projects, and the hint
// (TTY-only) stays off a piped stderr.
func TestList_HideTagsJSONStdoutClean(t *testing.T) {
	cfgPath := newHideTagsTestConfig(t)
	stdout, stderr, err := runCommandCapture(t, listCmd, []string{"list", "--config", cfgPath, "--json"})
	if err != nil {
		t.Fatalf("list --json: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("stdout is not clean JSON: %v\n%s", err, stdout)
	}
	if len(items) != 1 || items[0]["id"] != "p20260101a" {
		t.Errorf("got %v, want only p20260101a", items)
	}
	if stderr != "" {
		t.Errorf("stderr should be empty on a pipe, got %q", stderr)
	}
}

func TestList_AllHiddenMessage(t *testing.T) {
	cfg, tmp := newEditTestCfg(t)
	cfg.HideTags = []string{"archived"}
	mustMkProjectDir(t, cfg, "p20260101a")
	mustSeedSnapshot(t, cfg, "p20260101a", []string{"archived"})
	cfgPath := filepath.Join(tmp, "config.json")
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	out, err := runListCommand(t, []string{"list", "--config", cfgPath, "--no-color"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if want := "No visible projects. 1 hidden by hide_tags; use --no-hide."; !strings.Contains(out, want) {
		t.Errorf("got %q, want %q", out, want)
	}
}
