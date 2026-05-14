package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/metadata"
	"github.com/gorodulin/prj/internal/project"
	"github.com/gorodulin/prj/internal/text"
	"github.com/spf13/cobra"
)

// Edit-specific error codes. Shared codes live in cmd/bulkids.go.
const (
	errCurrentUnsupportedBulk = "current_unsupported_in_bulk"
	errTitleUnsupportedBulk   = "title_unsupported_in_bulk"
)

type editIntent struct {
	titleSet    *string
	tagsReplace *[]string
	tagsAdd     []string
	tagsRemove  []string
	force       bool
	dryRun      bool
}

type editResult struct {
	ID          string   `json:"id"`
	Status      string   `json:"status"`
	Title       string   `json:"title"`
	Path        string   `json:"path"`
	Local       bool     `json:"local"`
	Tags        []string `json:"tags"`
	TagsAdded   []string `json:"tags_added"`
	TagsRemoved []string `json:"tags_removed"`
}

var editCmd = &cobra.Command{
	Use:   "edit <project-id> [<project-id>...]",
	Short: "Edit project metadata",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runEdit,
}

func init() {
	rootCmd.AddCommand(editCmd)
	editCmd.Flags().String("title", "", "set project title (empty string clears) (single ID only)")
	editCmd.Flags().String("tags", "", "replace all tags (comma-separated, empty clears)")
	editCmd.Flags().String("tag", "", "deprecated alias for --tags")
	_ = editCmd.Flags().MarkDeprecated("tag", "use --tags instead")
	editCmd.Flags().String("add-tags", "", "add tags (comma-separated)")
	editCmd.Flags().String("remove-tags", "", "remove tags (comma-separated)")
	editCmd.Flags().Bool("force", false, "allow editing unknown project (creates metadata)")
	editCmd.Flags().Bool("dry-run", false, "compute changes without writing metadata")
	editCmd.Flags().Bool("json", false, `output as JSON envelope (suppresses stderr "no changes" line)`)
	editCmd.Flags().Bool("jsonl", false, "output as JSON Lines (one record per line)")
	editCmd.MarkFlagsMutuallyExclusive("json", "jsonl")
	editCmd.MarkFlagsMutuallyExclusive("tags", "tag")
}

func runEdit(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")
	jsonlOut, _ := cmd.Flags().GetBool("jsonl")

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return editError(jsonOut, jsonlOut, errConfigLoadFailed, fmt.Errorf("load config: %w", err).Error())
	}

	if err := requireConfig(cfg, "metadata_folder", "machine_name", "machine_id"); err != nil {
		return editError(jsonOut, jsonlOut, errConfigLoadFailed, err.Error())
	}

	deduped := dedupeStrings(args)
	bulk := len(deduped) > 1

	intent := buildIntent(cmd)
	if code, reason, ok := validateIntent(intent, bulk); !ok {
		return editError(jsonOut, jsonlOut, code, reason)
	}

	if !bulk {
		return runEditSingle(deduped[0], intent, cfg, jsonOut, jsonlOut)
	}
	return runEditBulk(deduped, intent, cfg, jsonOut, jsonlOut)
}

func runEditSingle(rawID string, intent editIntent, cfg config.Config, jsonOut, jsonlOut bool) error {
	id, err := expandProjectID(rawID, cfg)
	if err != nil {
		return editError(jsonOut, jsonlOut, errUnknownID, err.Error())
	}

	result, idErr := editProject(id, intent, cfg)
	if idErr != nil {
		if jsonOut {
			env := Envelope[editResult]{
				DryRun:  intent.dryRun,
				Results: []editResult{},
				Errors:  []IDError{*idErr},
			}
			_ = writeJSONEnvelope(env)
			os.Exit(1)
		}
		if jsonlOut {
			_ = writeJSONLines([]editResult{}, []IDError{*idErr})
			os.Exit(1)
		}
		return fmt.Errorf("%s", idErr.Reason)
	}

	if jsonOut {
		env := Envelope[editResult]{
			DryRun:  intent.dryRun,
			Results: []editResult{result},
			Errors:  []IDError{},
		}
		return writeJSONEnvelope(env)
	}
	if jsonlOut {
		return writeJSONLines([]editResult{result}, nil)
	}

	if intent.dryRun {
		fmt.Fprintln(os.Stderr, "DRY RUN — no metadata written")
	}
	switch result.Status {
	case "unchanged":
		fmt.Fprintln(os.Stderr, "no changes")
	default:
		if result.Title != "" {
			fmt.Printf("%s\t%s\n", result.ID, result.Title)
		} else {
			fmt.Println(result.ID)
		}
	}
	return nil
}

func runEditBulk(ids []string, intent editIntent, cfg config.Config, jsonOut, jsonlOut bool) error {
	var results []editResult
	var idErrors []IDError
	for _, id := range ids {
		if id == "current" {
			idErrors = append(idErrors, IDError{
				ID:     "current",
				Code:   errCurrentUnsupportedBulk,
				Reason: `the "current" virtual ID is not supported when editing multiple projects`,
			})
			continue
		}
		res, perr := editProject(id, intent, cfg)
		if perr != nil {
			idErrors = append(idErrors, *perr)
			continue
		}
		results = append(results, res)
	}

	if jsonOut {
		env := Envelope[editResult]{
			DryRun:  intent.dryRun,
			Results: results,
			Errors:  idErrors,
		}
		if err := writeJSONEnvelope(env); err != nil {
			return err
		}
		exitOnPartialFailure(len(idErrors))
		return nil
	}
	if jsonlOut {
		if err := writeJSONLines(results, idErrors); err != nil {
			return err
		}
		exitOnPartialFailure(len(idErrors))
		return nil
	}

	if intent.dryRun {
		fmt.Fprintln(os.Stderr, "DRY RUN — no metadata written")
	}
	for _, r := range results {
		fmt.Printf("%s\t%s\t%s\n", r.ID, r.Status, r.Title)
	}
	for _, e := range idErrors {
		fmt.Fprintf(os.Stderr, "Error: %s: %s\n", e.ID, e.Reason)
	}
	exitOnPartialFailure(len(idErrors))
	return nil
}

func buildIntent(cmd *cobra.Command) editIntent {
	intent := editIntent{}
	if cmd.Flags().Changed("title") {
		t := flagString(cmd, "title")
		intent.titleSet = &t
	}
	if cmd.Flags().Changed("tags") || cmd.Flags().Changed("tag") {
		var raw string
		if cmd.Flags().Changed("tags") {
			raw = flagString(cmd, "tags")
		} else {
			raw = flagString(cmd, "tag")
		}
		var tags []string
		if raw == "" {
			tags = []string{}
		} else {
			tags = text.ParseTags(raw)
		}
		intent.tagsReplace = &tags
	}
	if addRaw := flagString(cmd, "add-tags"); addRaw != "" {
		intent.tagsAdd = text.ParseTags(addRaw)
	}
	if remRaw := flagString(cmd, "remove-tags"); remRaw != "" {
		intent.tagsRemove = text.ParseTags(remRaw)
	}
	intent.force, _ = cmd.Flags().GetBool("force")
	intent.dryRun, _ = cmd.Flags().GetBool("dry-run")
	return intent
}

// validateIntent checks that the intent has at least one edit field and that
// flag combinations are coherent. Pure function; suitable for table tests.
// bulk=true rejects --title (not supported when editing multiple projects).
func validateIntent(intent editIntent, bulk bool) (code, reason string, ok bool) {
	hasAnyEdit := intent.titleSet != nil ||
		intent.tagsReplace != nil ||
		len(intent.tagsAdd) > 0 ||
		len(intent.tagsRemove) > 0
	if !hasAnyEdit {
		return errNoFlagsProvided, "nothing to edit: provide --title, --tags, --add-tags, or --remove-tags", false
	}
	if intent.tagsReplace != nil && (len(intent.tagsAdd) > 0 || len(intent.tagsRemove) > 0) {
		return errFlagsConflict, "--tags cannot be combined with --add-tags or --remove-tags", false
	}
	if bulk && intent.titleSet != nil {
		return errTitleUnsupportedBulk, "--title is not supported when editing multiple projects", false
	}
	return "", "", true
}

// editProject runs the edit pipeline for one resolved ID. Returns either a
// populated editResult or a non-nil *IDError. Used by both single-ID and
// bulk modes.
func editProject(id string, intent editIntent, cfg config.Config) (editResult, *IDError) {
	if cfg.ProjectIDType != "" && !project.IsValidID(id, cfg.ProjectIDType, cfg.ProjectIDPrefix) {
		return editResult{}, &IDError{
			ID:     id,
			Code:   errInvalidIDFormat,
			Reason: fmt.Sprintf("invalid project ID %q for format %s", id, cfg.ProjectIDType),
		}
	}

	metaDir := cfg.MetadataDir(id)
	projPath := filepath.Join(cfg.ProjectsFolder, id)

	folderExisted := false
	if cfg.ProjectsFolder != "" {
		if _, err := os.Stat(projPath); err == nil {
			folderExisted = true
		}
	}
	hasMetadata := false
	if _, err := os.Stat(metaDir); err == nil {
		hasMetadata = true
	}

	known := folderExisted || hasMetadata
	if !known && !intent.force {
		return editResult{}, &IDError{
			ID:     id,
			Code:   errUnknownID,
			Reason: fmt.Sprintf("unknown project %s (use --force to create metadata)", id),
		}
	}

	snapshots, err := metadata.ReadSnapshots(metaDir)
	if err != nil {
		return editResult{}, &IDError{
			ID:     id,
			Code:   errMetadataIOFailed,
			Reason: fmt.Sprintf("read metadata for %s: %v", id, err),
		}
	}
	heads := metadata.FindHeads(snapshots)
	current := metadata.LatestHead(snapshots)

	newTags := current.Tags
	if intent.tagsReplace != nil {
		newTags = *intent.tagsReplace
	} else {
		if len(intent.tagsAdd) > 0 {
			newTags = addToTags(newTags, intent.tagsAdd)
		}
		if len(intent.tagsRemove) > 0 {
			newTags = removeFromTags(newTags, intent.tagsRemove)
		}
	}

	titleSame := intent.titleSet == nil || (current.Title == *intent.titleSet)
	tagsSame := sliceEqual(current.Tags, newTags)

	displayTitle := current.Title
	if intent.titleSet != nil {
		displayTitle = *intent.titleSet
	}

	if titleSame && tagsSame {
		tags := current.Tags
		if tags == nil {
			tags = []string{}
		}
		return editResult{
			ID:          id,
			Status:      "unchanged",
			Title:       displayTitle,
			Path:        projPath,
			Local:       folderExisted,
			Tags:        tags,
			TagsAdded:   []string{},
			TagsRemoved: []string{},
		}, nil
	}

	tagsAdded, tagsRemoved := metadata.TagDeltas(current.Tags, newTags)

	var basedOn []string
	for _, h := range heads {
		basedOn = append(basedOn, h.Filename)
	}

	if !intent.dryRun {
		s := metadata.Snapshot{
			BasedOn:     basedOn,
			TitleSet:    intent.titleSet,
			Tags:        newTags,
			TagsAdded:   tagsAdded,
			TagsRemoved: tagsRemoved,
			MachineID:   cfg.MachineID,
			MachineName: cfg.MachineName,
			Version:     1,
		}

		if _, err := metadata.WriteSnapshot(metaDir, s); err != nil {
			return editResult{}, &IDError{
				ID:     id,
				Code:   errMetadataIOFailed,
				Reason: fmt.Sprintf("write metadata for %s: %v", id, err),
			}
		}

		if n, err := metadata.PurgeOldSnapshots(metaDir, cfg.RetentionDays); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: purge old snapshots for %s: %v\n", id, err)
		} else if n > 0 {
			fmt.Fprintf(os.Stderr, "Purged %d old snapshot(s) for %s\n", n, id)
		}
	}

	// folderExisted = true → updated, regardless of metadata.
	// folderExisted = false (force created metadata only) → created.
	status := "updated"
	if !folderExisted {
		status = "created"
	}

	if newTags == nil {
		newTags = []string{}
	}
	if tagsAdded == nil {
		tagsAdded = []string{}
	}
	if tagsRemoved == nil {
		tagsRemoved = []string{}
	}

	return editResult{
		ID:          id,
		Status:      status,
		Title:       displayTitle,
		Path:        projPath,
		Local:       folderExisted,
		Tags:        newTags,
		TagsAdded:   tagsAdded,
		TagsRemoved: tagsRemoved,
	}, nil
}

// editError formats a top-level error for the edit command. In non-JSON
// modes, returns a Go error so cobra's RunE printer handles it. In JSON
// mode, emits a top-level error envelope and exits 1. In JSONL mode,
// emits a single error line and exits 1.
func editError(jsonOut, jsonlOut bool, code, reason string) error {
	if jsonOut {
		env := Envelope[editResult]{
			Error:   &TopError{Code: code, Reason: reason},
			Results: []editResult{},
			Errors:  []IDError{},
		}
		_ = writeJSONEnvelope(env)
		os.Exit(1)
		return nil
	}
	if jsonlOut {
		_ = writeJSONLines[editResult](nil, []IDError{{Code: code, Reason: reason}})
		os.Exit(1)
		return nil
	}
	return fmt.Errorf("%s", reason)
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
