package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorodulin/prj/internal/config"
)

// newCurrentDefaultFixture builds a config with one project and returns the
// config path plus the project folder to chdir into.
func newCurrentDefaultFixture(t *testing.T) (cfgPath, projectDir string) {
	t.Helper()
	cfg, tmp := newEditTestCfg(t)
	mustMkProjectDir(t, cfg, "p20260101a")
	mustSeedSnapshot(t, cfg, "p20260101a", []string{"cli"})
	cfgPath = filepath.Join(tmp, "config.json")
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return cfgPath, filepath.Join(cfg.ProjectsFolder, "p20260101a")
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
}

func TestInfo_NoArgs_DefaultsToCurrent(t *testing.T) {
	cfgPath, projectDir := newCurrentDefaultFixture(t)
	chdir(t, projectDir)

	out, _, err := runCommandCapture(t, infoCmd, []string{"info", "--config", cfgPath, "--jsonl"})
	if err != nil {
		t.Fatalf("bare info inside project: %v", err)
	}
	if !strings.Contains(out, "p20260101a") {
		t.Errorf("bare info should resolve to current project, got: %q", out)
	}
}

func TestInfo_NoArgs_OutsideProject(t *testing.T) {
	cfgPath, projectDir := newCurrentDefaultFixture(t)
	chdir(t, filepath.Dir(filepath.Dir(projectDir))) // tmp root, outside projects folder

	_, _, err := runCommandCapture(t, infoCmd, []string{"info", "--config", cfgPath})
	if err == nil {
		t.Fatal("expected error for bare info outside a project folder, got nil")
	}
	if !strings.Contains(err.Error(), "not inside a project folder") {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestPath_NoArgs_DefaultsToCurrent(t *testing.T) {
	cfgPath, projectDir := newCurrentDefaultFixture(t)
	chdir(t, projectDir)

	out, _, err := runCommandCapture(t, pathCmd, []string{"path", "--config", cfgPath})
	if err != nil {
		t.Fatalf("bare path inside project: %v", err)
	}
	if !strings.Contains(strings.TrimSpace(out), "p20260101a") {
		t.Errorf("bare path should print current project folder, got: %q", out)
	}
}

func TestPath_NoArgs_OutsideProject(t *testing.T) {
	cfgPath, projectDir := newCurrentDefaultFixture(t)
	chdir(t, filepath.Dir(filepath.Dir(projectDir))) // tmp root, outside projects folder

	_, _, err := runCommandCapture(t, pathCmd, []string{"path", "--config", cfgPath})
	if err == nil {
		t.Fatal("expected error for bare path outside a project folder, got nil")
	}
	if !strings.Contains(err.Error(), "not inside a project folder") {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestEdit_NoArgs_DefaultsToCurrent(t *testing.T) {
	cfgPath, projectDir := newCurrentDefaultFixture(t)
	chdir(t, projectDir)

	out, errOut, err := runCommandCapture(t, editCmd, []string{"edit", "--config", cfgPath, "--add-tags", "newtag"})
	if err != nil {
		t.Fatalf("bare edit inside project: %v", err)
	}
	if !strings.Contains(out, "p20260101a") {
		t.Errorf("edit output should name the edited project, got: %q", out)
	}
	if !strings.Contains(errOut, "using current project p20260101a") {
		t.Errorf("defaulted edit should announce its target on stderr, got: %q", errOut)
	}
}

func TestEdit_NoArgs_JSONLSuppressesAnnouncement(t *testing.T) {
	cfgPath, projectDir := newCurrentDefaultFixture(t)
	chdir(t, projectDir)

	out, errOut, err := runCommandCapture(t, editCmd, []string{"edit", "--config", cfgPath, "--add-tags", "another", "--jsonl"})
	if err != nil {
		t.Fatalf("bare edit with --jsonl: %v", err)
	}
	if !strings.Contains(out, "p20260101a") {
		t.Errorf("jsonl output should name the edited project, got: %q", out)
	}
	if strings.Contains(errOut, "using current project") {
		t.Errorf("announcement must be suppressed in --jsonl mode, stderr: %q", errOut)
	}
}

func TestEdit_NoArgsNoFlags_NothingToEdit(t *testing.T) {
	cfgPath, projectDir := newCurrentDefaultFixture(t)
	chdir(t, projectDir)

	_, errOut, err := runCommandCapture(t, editCmd, []string{"edit", "--config", cfgPath})
	if err == nil {
		t.Fatal("expected 'nothing to edit' error, got nil")
	}
	if !strings.Contains(err.Error(), "nothing to edit") {
		t.Errorf("unexpected error: %q", err.Error())
	}
	if strings.Contains(errOut, "using current project") {
		t.Errorf("flag validation must fire before ID resolution, stderr: %q", errOut)
	}
}
