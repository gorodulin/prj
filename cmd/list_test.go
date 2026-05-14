package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/gorodulin/prj/internal/config"
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

// runListCommand invokes rootCmd with the given args, captures stdout, and
// resets globals/flag state so successive calls don't bleed into each other.
func runListCommand(t *testing.T, args []string) (string, error) {
	t.Helper()
	resetFlags := func(fs *pflag.FlagSet) {
		fs.VisitAll(func(f *pflag.Flag) {
			f.Changed = false
			_ = f.Value.Set(f.DefValue)
		})
	}
	resetFlags(rootCmd.PersistentFlags())
	resetFlags(listCmd.Flags())
	cfgFile = ""
	noColor = false

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	oldOut := os.Stdout
	os.Stdout = wOut

	rootCmd.SetArgs(args)
	execErr := rootCmd.Execute()

	_ = wOut.Close()
	os.Stdout = oldOut

	buf, _ := io.ReadAll(rOut)
	return string(buf), execErr
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
