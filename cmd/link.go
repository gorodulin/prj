package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/format"
	"github.com/gorodulin/prj/internal/linktree"
	"github.com/gorodulin/prj/internal/platform"
	"github.com/gorodulin/prj/internal/text"
)

var linkCmd = &cobra.Command{
	Use:   "link [project-id]",
	Short: "Sync project links in the links folder",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLink,
}

func init() {
	rootCmd.AddCommand(linkCmd)
	linkCmd.Flags().Bool("dry-run", false, "show changes without applying")
	linkCmd.Flags().Bool("verbose", false, "include unchanged links in output")
	linkCmd.Flags().String("kind", "", "override link kind (symlink, finder-alias)")
	linkCmd.Flags().Bool("warn-unplaced", false, "list projects with no placement")
	linkCmd.Flags().BoolP("all", "a", false, "include metadata-only projects (not present locally)")
}

func runLink(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := requireConfig(cfg, "links_folder", "projects_folder"); err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")
	warnUnplaced, _ := cmd.Flags().GetBool("warn-unplaced")
	showAll, _ := cmd.Flags().GetBool("all")

	linkKind := cfg.LinkKind
	if k := flagString(cmd, "kind"); k != "" {
		linkKind = k
	}
	if linkKind == "" {
		linkKind = config.LinkKindSymlink
	}
	if !config.IsValidLinkKind(linkKind) {
		return fmt.Errorf("unknown link kind %q (use %s)", linkKind, config.JoinQuoted(config.ValidLinkKinds))
	}

	titleFormat := cfg.LinkTitleFormat
	if titleFormat == "" {
		titleFormat = config.DefaultLinkTitleFormat
	}

	// Single-project filter.
	var filterID string
	if len(args) > 0 {
		filterID, err = expandProjectID(args[0], cfg)
		if err != nil {
			return err
		}
	}

	// 1. Build tree from links folder.
	tree, err := linktree.BuildTree(cfg.LinksFolder)
	if err != nil {
		return fmt.Errorf("build link tree: %w", err)
	}

	// 2. Collect all project IDs.
	ids, err := collectIDs(cfg)
	if err != nil {
		return err
	}

	// 3. Resolve projects and compute placements.
	//    folderPlacements groups projects by target folder for collision resolution.
	type projectPlacement struct {
		entry   linktree.ProjectEntry
		folders []*linktree.Folder
	}

	var placements []projectPlacement
	var unplaced []string
	// TODO(finder-comments): disabled until performance is resolved. See docs/finder-comments.md.
	setComments := false && cfg.LinkCommentFormat != ""
	var commentByID map[string][]byte
	if setComments {
		commentByID = make(map[string][]byte)
	}

	for _, id := range ids.all {
		if filterID != "" && id != filterID {
			continue
		}

		if !showAll && !ids.localSet[id] {
			continue // skip metadata-only projects
		}

		p := resolveProject(id, cfg, ids.localSet[id])
		tags := text.NormalizeTags(p.Tags)

		folders := linktree.FindPlacements(tree, tags, cfg.LinkSinkName)
		if len(folders) == 0 {
			unplaced = append(unplaced, id)
			continue
		}

		placements = append(placements, projectPlacement{
			entry:   linktree.ProjectEntry{ID: p.ID, Title: p.Title},
			folders: folders,
		})

		if setComments {
			commentByID[p.ID] = platform.EncodeBplistString(formatFinderComment(tags))
		}
	}

	// 4. Resolve link names per folder (handle collisions).
	//    Build desired state: linkPath → DesiredLink.
	folderProjects := make(map[string][]linktree.ProjectEntry) // folderPath → projects
	folderTargets := make(map[string]map[string]string)        // folderPath → (projectID → projectFolderPath)

	for _, pp := range placements {
		target := filepath.Join(cfg.ProjectsFolder, pp.entry.ID)
		for _, f := range pp.folders {
			fp := f.FullPath(cfg.LinksFolder)
			folderProjects[fp] = append(folderProjects[fp], pp.entry)
			if folderTargets[fp] == nil {
				folderTargets[fp] = make(map[string]string)
			}
			folderTargets[fp][pp.entry.ID] = target
		}
	}

	desired := make(map[string]linktree.DesiredLink)
	for fp, entries := range folderProjects {
		names := linktree.ResolveNames(entries, titleFormat, format.FuncMap(false))
		for _, e := range entries {
			linkPath := filepath.Join(fp, names[e.ID])
			desired[linkPath] = linktree.DesiredLink{
				Target: folderTargets[fp][e.ID],
				ID:     e.ID,
			}
		}
	}

	// 5. Scan actual state.
	actual, err := linktree.ScanManagedLinks(cfg.LinksFolder, cfg.ProjectsFolder, cfg.ProjectIDType, filterID)
	if err != nil {
		return fmt.Errorf("scan links: %w", err)
	}

	// 6. Reconcile.
	actions := linktree.Reconcile(desired, actual, linkKind)

	// 7. Print and apply.
	counts := printActions(actions, cfg.LinksFolder, verbose)

	if warnUnplaced && len(unplaced) > 0 {
		fmt.Fprintln(os.Stderr, "\nUnplaced projects:")
		for _, id := range unplaced {
			fmt.Fprintf(os.Stderr, "  %s\n", id)
		}
	}

	if dryRun {
		printSummary(counts, true)
		return nil
	}

	if err := linktree.Apply(actions, linkKind); err != nil {
		return fmt.Errorf("apply link changes: %w", err)
	}

	if setComments {
		updateLinkComments(actions, commentByID)
	}

	printSummary(counts, false)
	return nil
}

type actionCounts struct {
	created   int
	removed   int
	replaced  int
	conflicts int
	skipped   int
}

func printActions(actions []linktree.Action, linksRoot string, verbose bool) actionCounts {
	var c actionCounts
	for _, a := range actions {
		rel, _ := filepath.Rel(linksRoot, a.Path)
		if rel == "" {
			rel = a.Path
		}

		switch a.Kind {
		case linktree.ActionCreate:
			fmt.Printf("+ %s\t→ %s\n", rel, a.ID)
			c.created++
		case linktree.ActionRemove:
			fmt.Printf("- %s\t(%s)\n", rel, a.ID)
			c.removed++
		case linktree.ActionReplace:
			if a.NewPath != "" {
				newRel, _ := filepath.Rel(linksRoot, a.NewPath)
				if newRel == "" {
					newRel = a.NewPath
				}
				fmt.Printf("~ %s\t→ %s (%s)\n", newRel, a.ID, a.Detail)
			} else {
				fmt.Printf("~ %s\t→ %s (%s)\n", rel, a.ID, a.Detail)
			}
			c.replaced++
		case linktree.ActionConflict:
			fmt.Fprintf(os.Stderr, "! %s\t%s\n", rel, a.Detail)
			c.conflicts++
		case linktree.ActionSkip:
			if verbose {
				fmt.Printf("  %s\t→ %s\n", rel, a.ID)
			}
			c.skipped++
		}
	}
	return c
}

func formatFinderComment(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	parts := make([]string, len(tags))
	for i, t := range tags {
		parts[i] = "#" + t
	}
	return strings.Join(parts, " ")
}

func updateLinkComments(actions []linktree.Action, commentByID map[string][]byte) {
	var batch []platform.FinderComment

	for _, a := range actions {
		encoded := commentByID[a.ID]
		comment, _ := platform.DecodeBplistString(encoded)

		switch a.Kind {
		case linktree.ActionCreate, linktree.ActionReplace:
			commentPath := a.Path
			if a.NewPath != "" {
				commentPath = a.NewPath
			}
			batch = append(batch, platform.FinderComment{Path: commentPath, Comment: comment})
		case linktree.ActionSkip:
			changed, err := platform.FinderCommentChanged(a.Path, encoded)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: read comment on %s: %v\n", a.Path, err)
				continue
			}
			if changed {
				batch = append(batch, platform.FinderComment{Path: a.Path, Comment: comment})
			}
		}
	}

	if len(batch) == 0 {
		return
	}
	if err := platform.SetFinderComments(batch); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}
}

func printSummary(c actionCounts, dryRun bool) {
	prefix := ""
	if dryRun {
		prefix = "would: "
	}

	parts := []string{}
	if c.created > 0 {
		parts = append(parts, fmt.Sprintf("%d created", c.created))
	}
	if c.removed > 0 {
		parts = append(parts, fmt.Sprintf("%d removed", c.removed))
	}
	if c.replaced > 0 {
		parts = append(parts, fmt.Sprintf("%d replaced", c.replaced))
	}
	if c.conflicts > 0 {
		parts = append(parts, fmt.Sprintf("%d conflict(s)", c.conflicts))
	}

	if len(parts) == 0 {
		fmt.Printf("\n%sno changes\n", prefix)
		return
	}

	fmt.Printf("\n%s", prefix)
	for i, p := range parts {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(p)
	}
	fmt.Println()
}
