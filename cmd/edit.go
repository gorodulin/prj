package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/metadata"
	"github.com/gorodulin/prj/internal/project"
	"github.com/gorodulin/prj/internal/text"
)

var editCmd = &cobra.Command{
	Use:   "edit <project-id>",
	Short: "Edit project metadata",
	Args:  cobra.ExactArgs(1),
	RunE:  runEdit,
}

func init() {
	rootCmd.AddCommand(editCmd)
	editCmd.Flags().String("title", "", "set project title (empty string clears)")
	editCmd.Flags().String("tags", "", "replace all tags (comma-separated, empty clears)")
	editCmd.Flags().String("add-tags", "", "add tags (comma-separated)")
	editCmd.Flags().String("remove-tags", "", "remove tags (comma-separated)")
	editCmd.Flags().Bool("force", false, "allow editing unknown project (creates metadata)")
}

func runEdit(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := requireConfig(cfg, "metadata_folder", "machine_name", "machine_id"); err != nil {
		return err
	}

	id, err := expandProjectID(args[0], cfg)
	if err != nil {
		return err
	}

	// Validate ID format if configured.
	if cfg.ProjectIDType != "" && !project.IsValidID(id, cfg.ProjectIDType, cfg.ProjectIDPrefix) {
		return fmt.Errorf("invalid project ID %q for format %s", id, cfg.ProjectIDType)
	}

	// Validate flag combinations.
	titleChanged := cmd.Flags().Changed("title")
	tagsChanged := cmd.Flags().Changed("tags")
	addTagsRaw := flagString(cmd, "add-tags")
	removeTagsRaw := flagString(cmd, "remove-tags")
	hasAddTags := addTagsRaw != ""
	hasRemoveTags := removeTagsRaw != ""

	if !titleChanged && !tagsChanged && !hasAddTags && !hasRemoveTags {
		return fmt.Errorf("nothing to edit: provide --title, --tags, --add-tags, or --remove-tags")
	}
	if tagsChanged && (hasAddTags || hasRemoveTags) {
		return fmt.Errorf("--tags cannot be combined with --add-tags or --remove-tags")
	}

	force, _ := cmd.Flags().GetBool("force")
	metaDir := cfg.MetadataDir(id)

	// Check if project is known (has folder or metadata).
	known := false
	if cfg.ProjectsFolder != "" {
		if _, err := os.Stat(filepath.Join(cfg.ProjectsFolder, id)); err == nil {
			known = true
		}
	}
	if !known {
		if _, err := os.Stat(metaDir); err == nil {
			known = true
		}
	}
	if !known && !force {
		return fmt.Errorf("unknown project %s (use --force to create metadata)", id)
	}

	// Read existing snapshots and resolve current state.
	snapshots, err := metadata.ReadSnapshots(metaDir)
	if err != nil {
		return fmt.Errorf("read metadata for %s: %w", id, err)
	}
	heads := metadata.FindHeads(snapshots)
	current := metadata.LatestHead(snapshots)

	// Compute new title.
	var titleSet *string
	if titleChanged {
		t := flagString(cmd, "title")
		titleSet = &t
	}

	// Compute new tags.
	newTags := current.Tags
	if tagsChanged {
		raw := flagString(cmd, "tags")
		if raw == "" {
			newTags = []string{}
		} else {
			newTags = text.ParseTags(raw)
		}
	} else {
		if hasAddTags {
			newTags = addToTags(newTags, text.ParseTags(addTagsRaw))
		}
		if hasRemoveTags {
			newTags = removeFromTags(newTags, text.ParseTags(removeTagsRaw))
		}
	}

	// Skip write if nothing actually changed.
	titleSame := titleSet == nil || (current.Title == *titleSet)
	tagsSame := sliceEqual(current.Tags, newTags)
	if titleSame && tagsSame {
		fmt.Fprintln(os.Stderr, "no changes")
		return nil
	}

	tagsAdded, tagsRemoved := metadata.TagDeltas(current.Tags, newTags)

	// Build based_on from current heads.
	var basedOn []string
	for _, h := range heads {
		basedOn = append(basedOn, h.Filename)
	}

	s := metadata.Snapshot{
		BasedOn:     basedOn,
		TitleSet:    titleSet,
		Tags:        newTags,
		TagsAdded:   tagsAdded,
		TagsRemoved: tagsRemoved,
		MachineID:   cfg.MachineID,
		MachineName: cfg.MachineName,
		Version:     1,
	}

	if _, err := metadata.WriteSnapshot(metaDir, s); err != nil {
		return fmt.Errorf("write metadata for %s: %w", id, err)
	}

	if n, err := metadata.PurgeOldSnapshots(metaDir, cfg.RetentionDays); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: purge old snapshots for %s: %v\n", id, err)
	} else if n > 0 {
		fmt.Fprintf(os.Stderr, "Purged %d old snapshot(s) for %s\n", n, id)
	}

	// Resolve display title: explicit > inherited > empty.
	title := current.Title
	if titleSet != nil {
		title = *titleSet
	}

	if title != "" {
		fmt.Printf("%s\t%s\n", id, title)
	} else {
		fmt.Println(id)
	}

	return nil
}

func sliceEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// addToTags merges additional tags into the existing set.
func addToTags(existing, add []string) []string {
	seen := make(map[string]bool, len(existing))
	for _, t := range existing {
		seen[t] = true
	}
	merged := append([]string(nil), existing...)
	for _, t := range add {
		if !seen[t] {
			seen[t] = true
			merged = append(merged, t)
		}
	}
	sort.Strings(merged)
	return merged
}

// removeFromTags removes specified tags from the existing set.
func removeFromTags(existing, remove []string) []string {
	drop := make(map[string]bool, len(remove))
	for _, t := range remove {
		drop[t] = true
	}
	var result []string
	for _, t := range existing {
		if !drop[t] {
			result = append(result, t)
		}
	}
	if result == nil {
		return []string{}
	}
	return result
}
