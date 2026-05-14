package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/metadata"
	"github.com/gorodulin/prj/internal/project"
)

// newNewTestConfig writes a minimal config sufficient for `prj new` to
// produce a metadata snapshot (i.e. with metadata_folder, machine_*).
func newNewTestConfig(t *testing.T) string {
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
	cfgPath := filepath.Join(tmp, "config.json")
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return cfgPath
}

func runNewCommand(t *testing.T, args []string) (string, error) {
	t.Helper()
	resetFlags := func(fs *pflag.FlagSet) {
		fs.VisitAll(func(f *pflag.Flag) {
			f.Changed = false
			_ = f.Value.Set(f.DefValue)
		})
	}
	resetFlags(rootCmd.PersistentFlags())
	resetFlags(newCmd.Flags())
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

func TestNew_TagAlias(t *testing.T) {
	cfgPath := newNewTestConfig(t)

	out, err := runNewCommand(t, []string{"new", "--config", cfgPath, "--title", "Alias", "--tag", "wip,cli"})
	if err != nil {
		t.Fatalf("new --tag: %v", err)
	}
	parts := strings.SplitN(strings.TrimRight(out, "\n"), "\t", 2)
	if len(parts) != 2 {
		t.Fatalf("expected '<id>\\t<path>' output, got %q", out)
	}
	id := parts[0]

	// Find a config to load and confirm the snapshot's tags.
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	snaps, err := metadata.ReadSnapshots(cfg.MetadataDir(id))
	if err != nil {
		t.Fatalf("read snapshots: %v", err)
	}
	if len(snaps) == 0 {
		t.Fatalf("no snapshots written for %s", id)
	}
	head := metadata.LatestHead(snaps)
	want := []string{"cli", "wip"}
	if !sliceEqual(head.Tags, want) {
		t.Errorf("tags = %v, want %v", head.Tags, want)
	}
}

func TestNew_TagsTagMutuallyExclusive(t *testing.T) {
	cfgPath := newNewTestConfig(t)
	_, err := runNewCommand(t, []string{"new", "--config", cfgPath, "--title", "X", "--tags", "a", "--tag", "b"})
	if err == nil {
		t.Fatal("expected mutual-exclusion error, got nil")
	}
	if !strings.Contains(err.Error(), "none of the others can be") {
		t.Errorf("error %q is not cobra's mutual-exclusion error", err.Error())
	}
}
