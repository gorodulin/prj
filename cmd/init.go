package cmd

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/format"
	"github.com/gorodulin/prj/internal/platform"
	"github.com/gorodulin/prj/internal/project"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard",
	Long:  "Walk through initial prj configuration interactively.",
	Args:  cobra.NoArgs,
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, _ []string) error {
	if !format.IsTTY(os.Stdin) {
		return fmt.Errorf("requires an interactive terminal; use 'prj config set <key> <value>' to configure manually")
	}

	cfgPath, err := configPath()
	if err != nil {
		return err
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	color := format.ResolveColor(os.Stderr, noColor, cfg.Color)
	p := &prompter{r: bufio.NewReader(os.Stdin), color: color, out: os.Stderr}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		fmt.Fprintln(os.Stderr, "\nInterrupted. Progress saved.")
		os.Exit(0)
	}()
	defer signal.Stop(sigs)

	fmt.Fprintln(p.out, "prj setup wizard — press Enter to keep the current value.")

	if err := wizardMachineIdentity(p, cfgPath, &cfg); err != nil {
		return err
	}
	if err := wizardProjectsFolder(p, cfgPath, &cfg); err != nil {
		return err
	}
	if err := wizardMetadata(p, cfgPath, &cfg); err != nil {
		return err
	}
	if err := wizardLinks(p, cfgPath, &cfg); err != nil {
		return err
	}

	fmt.Fprintln(p.out, "\nDone. Run 'prj config list' to review your settings.")
	return nil
}

// ── wizard groups ─────────────────────────────────────────────────────────────

func wizardMachineIdentity(p *prompter, cfgPath string, cfg *config.Config) error {
	fmt.Fprintln(p.out)

	defName := cfg.MachineName
	if defName == "" {
		defName, _ = os.Hostname()
	}
	name, err := p.readLine("machine_name", defName)
	if err != nil {
		return err
	}
	if name != cfg.MachineName {
		if err := saveField(cfgPath, "machine_name", name, p); err != nil {
			return err
		}
		cfg.MachineName = name
	}

	defID := cfg.MachineID
	if defID == "" {
		if defID, err = generateMachineID(); err != nil {
			return err
		}
	}
	for {
		id, err := p.readLine("machine_id", defID)
		if err != nil {
			return err
		}
		tmp := *cfg
		tmp.MachineID = id
		if err := tmp.Validate(); err != nil {
			fmt.Fprintf(p.out, "  %s\n", trimConfigPrefix(err))
			continue
		}
		if id != cfg.MachineID {
			if err := saveField(cfgPath, "machine_id", id, p); err != nil {
				return err
			}
			cfg.MachineID = id
		}
		break
	}

	return nil
}

func wizardProjectsFolder(p *prompter, cfgPath string, cfg *config.Config) error {
	fmt.Fprintln(p.out)

	prevFolder := cfg.ProjectsFolder

	validateProjects := func(path string) error {
		tmp := *cfg
		tmp.ProjectsFolder = path
		return tmp.Validate()
	}

	var folder string
	for {
		var err error
		folder, err = p.readPath("projects_folder", cfg.ProjectsFolder, "Select projects folder", true, validateProjects)
		if err != nil {
			return err
		}
		if folder == "" {
			fmt.Fprintln(p.out, "  projects_folder is required.")
			continue
		}
		ok, err := p.offerMkdir(folder)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if folder != cfg.ProjectsFolder {
			if err := saveField(cfgPath, "projects_folder", folder, p); err != nil {
				return err
			}
			cfg.ProjectsFolder = folder
		}
		break
	}

	// Detect project_id_type from folder contents when appropriate.
	folderChanged := folder != prevFolder
	shouldScan := folderChanged || cfg.ProjectIDType == ""
	var detectedType, detectedPrefix, detectedExample string
	if shouldScan {
		detectedType, detectedPrefix, detectedExample, _ = detectIDFormat(folder)
	}

	defType := cfg.ProjectIDType
	if defType == "" {
		defType = config.DefaultProjectIDType
	}
	if shouldScan && detectedType != "" {
		defType = detectedType
		fmt.Fprintf(p.out, p.dim("  detected %s (e.g. %s)\n"), detectedType, detectedExample)
	}

	idType, err := p.readEnum("project_id_type", defType, config.ValidProjectIDTypes)
	if err != nil {
		return err
	}
	if idType != cfg.ProjectIDType {
		if err := saveField(cfgPath, "project_id_type", idType, p); err != nil {
			return err
		}
		cfg.ProjectIDType = idType
	}

	if idType == project.FormatAYMDb {
		defPrefix := cfg.ProjectIDPrefix
		if defPrefix == "" {
			if detectedPrefix != "" {
				defPrefix = detectedPrefix
			} else {
				defPrefix = config.DefaultProjectIDPrefix
			}
		}
		for {
			prefix, err := p.readLine("project_id_prefix", defPrefix)
			if err != nil {
				return err
			}
			if !project.IsValidPrefix(prefix) {
				fmt.Fprintf(p.out, "  Must be 1-5 lowercase letters (got %q).\n", prefix)
				continue
			}
			if prefix != cfg.ProjectIDPrefix {
				if err := saveField(cfgPath, "project_id_prefix", prefix, p); err != nil {
					return err
				}
				cfg.ProjectIDPrefix = prefix
			}
			break
		}
	}

	return nil
}

func wizardMetadata(p *prompter, cfgPath string, cfg *config.Config) error {
	fmt.Fprintln(p.out)

	validateMetadata := func(path string) error {
		tmp := *cfg
		tmp.MetadataFolder = path
		return tmp.Validate()
	}

	for {
		folder, err := p.readPath("metadata_folder", cfg.MetadataFolder, "Select metadata folder", true, validateMetadata)
		if err != nil {
			return err
		}
		if folder == "" {
			fmt.Fprintln(p.out, "  metadata_folder is required.")
			continue
		}
		ok, err := p.offerMkdir(folder)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if folder != cfg.MetadataFolder {
			if err := saveField(cfgPath, "metadata_folder", folder, p); err != nil {
				return err
			}
			cfg.MetadataFolder = folder
		}
		break
	}

	return nil
}

func wizardLinks(p *prompter, cfgPath string, cfg *config.Config) error {
	fmt.Fprintln(p.out)

	validateLinks := func(path string) error {
		tmp := *cfg
		tmp.LinksFolder = path
		return tmp.Validate()
	}

	for {
		folder, err := p.readPath("links_folder", cfg.LinksFolder, "Select links folder", true, validateLinks)
		if err != nil {
			return err
		}
		if folder == "" {
			fmt.Fprintln(p.out, "  links_folder is required.")
			continue
		}
		ok, err := p.offerMkdir(folder)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if folder != cfg.LinksFolder {
			if err := saveField(cfgPath, "links_folder", folder, p); err != nil {
				return err
			}
			cfg.LinksFolder = folder
		}
		break
	}

	defKind := cfg.LinkKind
	if defKind == "" {
		defKind = platform.DefaultLinkKind()
	}
	kinds := platform.SupportedLinkTypes()
	if len(kinds) > 1 {
		kind, err := p.readEnum("link_kind", defKind, kinds)
		if err != nil {
			return err
		}
		if kind != cfg.LinkKind {
			if err := saveField(cfgPath, "link_kind", kind, p); err != nil {
				return err
			}
			cfg.LinkKind = kind
		}
	}

	defSink := cfg.LinkSinkName
	if defSink == "" {
		defSink = "_unsorted"
	}
	sink, err := p.readLine("link_sink_name", defSink)
	if err != nil {
		return err
	}
	if sink != cfg.LinkSinkName {
		if err := saveField(cfgPath, "link_sink_name", sink, p); err != nil {
			return err
		}
		cfg.LinkSinkName = sink
	}

	return nil
}

// ── prompter ──────────────────────────────────────────────────────────────────

type prompter struct {
	r     *bufio.Reader
	color bool
	out   io.Writer
}

func (p *prompter) dim(s string) string {
	if p.color {
		return "\033[2m" + s + "\033[0m"
	}
	return s
}

// readLine prints a labelled prompt with def as the bracketed default.
// An empty response returns def unchanged.
func (p *prompter) readLine(label, def string) (string, error) {
	fmt.Fprintf(p.out, "  %s [%s]: ", label, p.dim(def))
	line, err := p.r.ReadString('\n')
	if err != nil && err != io.EOF {
		return def, err
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return def, nil
	}
	return line, nil
}

// readEnum shows a numbered menu. An empty response or invalid input keeps def.
func (p *prompter) readEnum(label, def string, choices []string) (string, error) {
	fmt.Fprintf(p.out, "  %s:\n", label)
	for i, c := range choices {
		mark := " "
		if c == def {
			mark = "*"
		}
		fmt.Fprintf(p.out, "    %s [%d] %s\n", mark, i+1, c)
	}
	for {
		fmt.Fprintf(p.out, "  choice [%s]: ", p.dim(def))
		line, err := p.r.ReadString('\n')
		if err != nil && err != io.EOF {
			return def, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return def, nil
		}
		for i, c := range choices {
			if line == fmt.Sprintf("%d", i+1) || line == c {
				return c, nil
			}
		}
		fmt.Fprintf(p.out, "  Enter 1-%d or the exact name.\n", len(choices))
	}
}

// readConfirm asks a Y/n or y/N question; empty input returns defYes.
func (p *prompter) readConfirm(label string, defYes bool) (bool, error) {
	prompt := "[Y/n]"
	if !defYes {
		prompt = "[y/N]"
	}
	fmt.Fprintf(p.out, "  %s %s: ", label, prompt)
	line, err := p.r.ReadString('\n')
	if err != nil && err != io.EOF {
		return defYes, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defYes, nil
	}
	return line == "y" || line == "yes", nil
}

// readPath prompts for an absolute path. Blank Enter keeps def (if set).
// When def is empty: required=true opens the picker, required=false skips (returns "").
// Typing "?" always opens the picker. Typing "skip" returns "".
// validate (optional) is called before confirming any candidate path.
func (p *prompter) readPath(label, def, pickerTitle string, required bool, validate func(string) error) (string, error) {
	for {
		hint := p.dim(" (? to browse)")
		if def == "" && !required {
			hint = p.dim(" (? to browse, Enter to skip)")
		}
		fmt.Fprintf(p.out, "  %s [%s]%s: ", label, p.dim(def), hint)
		line, err := p.r.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		line = strings.TrimSpace(line)

		if line == "" {
			if def != "" {
				return def, nil
			}
			if !required {
				return "", nil // skip optional field
			}
			line = "?" // required with no value → open picker
		}

		if line == "?" {
			if picked, ok := guiPickFolder(pickerTitle, def); ok {
				path := normalizePath(picked)
				fmt.Fprintf(p.out, "  → %s\n", path)
				if validate != nil {
					if err := validate(path); err != nil {
						fmt.Fprintf(p.out, "  %s\n", trimConfigPrefix(err))
						continue
					}
				}
				keep, _ := p.readConfirm("Use this path?", true)
				if keep {
					return path, nil
				}
			} else {
				fmt.Fprintln(p.out, p.dim("  (no folder picker available — type the path)"))
			}
			continue
		}

		if strings.ToLower(line) == "skip" {
			return "", nil
		}

		path := normalizePath(line)
		if !filepath.IsAbs(path) {
			fmt.Fprintf(p.out, "  Must be absolute path (got %q). Try again.\n", path)
			continue
		}
		if validate != nil {
			if err := validate(path); err != nil {
				fmt.Fprintf(p.out, "  %s\n", trimConfigPrefix(err))
				continue
			}
		}
		return path, nil
	}
}

// offerMkdir asks to create path if it does not exist.
// Returns ok=false when the path does not exist and the user declined to create it.
func (p *prompter) offerMkdir(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil
	}
	create, err := p.readConfirm(fmt.Sprintf("%q does not exist. Create it?", path), true)
	if err != nil {
		return false, err
	}
	if !create {
		return false, nil
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return false, fmt.Errorf("mkdir %s: %w", path, err)
	}
	fmt.Fprintln(p.out, p.dim("  ✓ directory created"))
	return true, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func saveField(cfgPath, key, value string, p *prompter) error {
	if err := config.SetField(cfgPath, key, value); err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	fmt.Fprintln(p.out, p.dim(fmt.Sprintf("  ✓ %s = %s", key, value)))
	return nil
}

func normalizePath(s string) string {
	if strings.HasPrefix(s, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			s = filepath.Join(home, s[2:])
		}
	}
	return filepath.Clean(s)
}

func generateMachineID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// detectIDFormat scans dir and returns the dominant project ID format.
// Returns found=false when there is no clear winner (< 60% match rate).
func detectIDFormat(dir string) (idType, prefix, example string, found bool) {
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return "", "", "", false
	}

	type stats struct{ count int; example string }
	fmtStats := map[string]*stats{}
	prefixVotes := map[string]int{}
	total := 0

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		total++

		matched := false
		for _, f := range []string{project.FormatUUIDv7, project.FormatULID, project.FormatKSUID} {
			if project.IsValidID(name, f, "") {
				if fmtStats[f] == nil {
					fmtStats[f] = &stats{}
				}
				fmtStats[f].count++
				if fmtStats[f].example == "" {
					fmtStats[f].example = name
				}
				matched = true
				break
			}
		}
		if !matched {
			if pfx := extractAYMDbPrefix(name); pfx != "" && project.IsValidID(name, project.FormatAYMDb, pfx) {
				f := project.FormatAYMDb
				if fmtStats[f] == nil {
					fmtStats[f] = &stats{}
				}
				fmtStats[f].count++
				if fmtStats[f].example == "" {
					fmtStats[f].example = name
				}
				prefixVotes[pfx]++
			}
		}
	}

	if total == 0 {
		return "", "", "", false
	}

	bestFmt, bestCount := "", 0
	for f, s := range fmtStats {
		if s.count > bestCount {
			bestCount = s.count
			bestFmt = f
		}
	}
	if bestFmt == "" || float64(bestCount)/float64(total) < 0.6 {
		return "", "", "", false
	}

	bestPfx, bestPfxCount := "", 0
	if bestFmt == project.FormatAYMDb {
		for pfx, c := range prefixVotes {
			if c > bestPfxCount {
				bestPfxCount = c
				bestPfx = pfx
			}
		}
	}

	return bestFmt, bestPfx, fmtStats[bestFmt].example, true
}

// extractAYMDbPrefix returns the prefix portion of a potential aYYYYMMDDb name.
func extractAYMDbPrefix(name string) string {
	for i, c := range name {
		if c >= '0' && c <= '9' {
			pfx := name[:i]
			if project.IsValidPrefix(pfx) {
				return pfx
			}
			return ""
		}
	}
	return ""
}

// trimConfigPrefix removes the leading "invalid config: " from SetField errors
// so they display cleanly inline.
func trimConfigPrefix(err error) string {
	return strings.TrimPrefix(err.Error(), "invalid config: ")
}
