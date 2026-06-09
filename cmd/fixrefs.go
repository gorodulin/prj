package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/project"
	"github.com/gorodulin/prj/internal/projectref"
	"github.com/spf13/cobra"
)

var fixrefsCmd = &cobra.Command{
	Use:   "fixrefs <project-id> [<project-id>...]",
	Short: "Rewrite cross-project symlinks as relative paths",
	Long: `Scan project folders for symlinks that reference a sibling project and
rewrite their targets as machine-independent relative paths.

A symlink is a candidate when its target contains exactly one path component
that is a valid project ID (for the configured format). The portion before the
ID is rebuilt as a relative path; the suffix after the ID is preserved. The
target need not exist.

Default is a dry-run preview; pass --apply to rewrite. "current" resolves to the
project containing the working directory (single ID only).`,
	Args: cobra.MinimumNArgs(1),
	RunE: runFixrefs,
}

func init() {
	rootCmd.AddCommand(fixrefsCmd)
	fixrefsCmd.Flags().Bool("apply", false, "rewrite symlinks (default: preview only)")
	fixrefsCmd.Flags().Bool("verbose", false, "also list unchanged symlinks")
}

// fixrefsLinkReport mirrors what a future --json refChange would expose. The
// command assembles these, then renders them separately, so adding JSON later
// is a serialization step rather than a refactor.
type fixrefsLinkReport struct {
	path     string
	old, new string
	targetID string
	changed  bool
	applied  bool
	err      error
}

type fixrefsProjectReport struct {
	id    string
	path  string
	links []fixrefsLinkReport
}

func runFixrefs(cmd *cobra.Command, args []string) error {
	apply, _ := cmd.Flags().GetBool("apply")
	verbose, _ := cmd.Flags().GetBool("verbose")

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := requireConfig(cfg, "projects_folder"); err != nil {
		return err
	}
	if cfg.ProjectIDType == "" {
		return fmt.Errorf("fixrefs requires project_id_type to be set (run: prj config set project_id_type <type>)")
	}

	projectsFolder, err := filepath.Abs(cfg.ProjectsFolder)
	if err != nil {
		return fmt.Errorf("resolve projects folder: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	ids := dedupeStrings(args)
	bulk := len(ids) > 1

	var reports []fixrefsProjectReport
	dots := &dotPrinter{}
	failed := false

	for _, rawID := range ids {
		id, rerr := resolveFixrefsID(rawID, cfg, bulk)
		if rerr != nil {
			fmt.Fprintf(os.Stderr, "Error: %s: %s\n", rawID, rerr)
			failed = true
			continue
		}

		projDir := filepath.Join(projectsFolder, id)
		if fi, serr := os.Stat(projDir); serr != nil || !fi.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: %s: project folder not found locally\n", id)
			failed = true
			continue
		}

		cands, interrupted, serr := projectref.Scan(ctx, projDir, projectsFolder, cfg.ProjectIDType, cfg.ProjectIDPrefix, dots.tick)
		if serr != nil {
			fmt.Fprintf(os.Stderr, "Error: %s: %v\n", id, serr)
			failed = true
			continue
		}

		if interrupted {
			// Abort before any mutation; report what was scanned.
			dots.done()
			reports = append(reports, fixrefsProjectReport{
				id:    id,
				path:  projDir,
				links: buildLinkReports(cands, nil),
			})
			renderFixrefsHuman(reports, projectsFolder, verbose, apply)
			fmt.Fprintln(os.Stderr, "\nInterrupted.")
			os.Exit(1)
		}

		var outcomes []projectref.Outcome
		if apply {
			outcomes = projectref.Apply(cands)
		}
		reports = append(reports, fixrefsProjectReport{
			id:    id,
			path:  projDir,
			links: buildLinkReports(cands, outcomes),
		})
	}
	dots.done()

	renderFixrefsHuman(reports, projectsFolder, verbose, apply)
	if failed {
		os.Exit(1)
	}
	return nil
}

// resolveFixrefsID expands "current", rejects it in bulk mode (mirrors edit),
// and validates the ID against the configured format.
func resolveFixrefsID(rawID string, cfg config.Config, bulk bool) (string, error) {
	if rawID == "current" && bulk {
		return "", fmt.Errorf(`the "current" virtual ID is not supported when scanning multiple projects`)
	}
	id, err := expandProjectID(rawID, cfg)
	if err != nil {
		return "", err
	}
	if !project.IsValidID(id, cfg.ProjectIDType, cfg.ProjectIDPrefix) {
		return "", fmt.Errorf("%q is not a valid project ID (expected %s format)", id, cfg.ProjectIDType)
	}
	return id, nil
}

// buildLinkReports joins scan candidates with apply outcomes (matched by path)
// into the renderable per-link view. outcomes may be nil (dry-run).
func buildLinkReports(cands []projectref.Candidate, outcomes []projectref.Outcome) []fixrefsLinkReport {
	byPath := make(map[string]projectref.Outcome, len(outcomes))
	for _, o := range outcomes {
		byPath[o.Path] = o
	}
	out := make([]fixrefsLinkReport, 0, len(cands))
	for _, c := range cands {
		lr := fixrefsLinkReport{
			path:     c.Path,
			old:      c.RawTarget,
			new:      c.WantTarget,
			targetID: c.TargetID,
			changed:  c.NeedsChange,
		}
		if o, ok := byPath[c.Path]; ok {
			lr.applied = o.Applied
			lr.err = o.Err
		}
		out = append(out, lr)
	}
	return out
}

func renderFixrefsHuman(reports []fixrefsProjectReport, projectsFolder string, verbose, apply bool) {
	var relinked, unchanged, failed int
	for _, rep := range reports {
		for _, lr := range rep.links {
			rel, relErr := filepath.Rel(projectsFolder, lr.path)
			if relErr != nil || rel == "" {
				rel = lr.path
			}
			switch {
			case lr.err != nil:
				fmt.Fprintf(os.Stderr, "! %s\t%v\n", rel, lr.err)
				failed++
			case lr.changed:
				fmt.Printf("~ %s\t%s → %s\n", rel, lr.old, lr.new)
				relinked++
			default:
				if verbose {
					fmt.Printf("  %s\t%s\n", rel, lr.old)
				}
				unchanged++
			}
		}
	}
	printFixrefsSummary(relinked, unchanged, failed, apply)
}

func printFixrefsSummary(relinked, unchanged, failed int, apply bool) {
	prefix := "would: "
	if apply {
		prefix = ""
	}

	if relinked == 0 && failed == 0 {
		if unchanged == 0 {
			fmt.Println("no project-ref symlinks found")
		} else {
			fmt.Printf("%sno changes (%d unchanged)\n", prefix, unchanged)
		}
		return
	}

	parts := []string{fmt.Sprintf("%d relinked", relinked)}
	if unchanged > 0 {
		parts = append(parts, fmt.Sprintf("%d unchanged", unchanged))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	fmt.Printf("%s%s\n", prefix, strings.Join(parts, ", "))
}

// dotPrinter emits one "." to stderr per directory scanned, then a closing
// newline. Presentation stays in cmd; the package signals progress via callback.
type dotPrinter struct {
	printed bool
}

func (d *dotPrinter) tick() {
	fmt.Fprint(os.Stderr, ".")
	d.printed = true
}

func (d *dotPrinter) done() {
	if d.printed {
		fmt.Fprintln(os.Stderr)
		d.printed = false
	}
}
