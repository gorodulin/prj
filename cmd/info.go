package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/project"
	"github.com/spf13/cobra"
)

type infoResult struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Path        string   `json:"path"`
	Tags        []string `json:"tags"`
	Local       bool     `json:"local"`
	HasReadme   bool     `json:"has_readme"`
	Date        string   `json:"date,omitempty"`
	DateTimeUTC string   `json:"datetime_utc,omitempty"`
}

var infoCmd = &cobra.Command{
	Use:   "info <project-id> [<project-id>...]",
	Short: "Display information about a project",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().Bool("json", false, "output as JSON envelope")
	infoCmd.Flags().Bool("jsonl", false, "output as JSON Lines (one record per line)")
	infoCmd.MarkFlagsMutuallyExclusive("json", "jsonl")
}

func runInfo(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")
	jsonlOut, _ := cmd.Flags().GetBool("jsonl")

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return infoError(jsonOut, jsonlOut, errConfigLoadFailed, fmt.Errorf("load config: %w", err).Error())
	}

	if err := requireConfig(cfg, "projects_folder"); err != nil {
		return infoError(jsonOut, jsonlOut, errConfigLoadFailed, err.Error())
	}

	deduped := dedupeStrings(args)
	if len(deduped) == 1 {
		return runInfoSingle(deduped[0], cfg, jsonOut, jsonlOut)
	}
	return runInfoBulk(deduped, cfg, jsonOut, jsonlOut)
}

func runInfoSingle(rawID string, cfg config.Config, jsonOut, jsonlOut bool) error {
	id, err := expandProjectID(rawID, cfg)
	if err != nil {
		return infoError(jsonOut, jsonlOut, errUnknownID, err.Error())
	}

	res, idErr := infoProject(id, cfg)
	if idErr != nil {
		if jsonOut {
			env := Envelope[infoResult]{
				Results: []infoResult{},
				Errors:  []IDError{*idErr},
			}
			_ = writeJSONEnvelope(env)
			os.Exit(1)
		}
		if jsonlOut {
			_ = writeJSONLines[infoResult](nil, []IDError{*idErr})
			os.Exit(1)
		}
		return fmt.Errorf("%s", idErr.Reason)
	}

	if jsonOut {
		env := Envelope[infoResult]{
			Results: []infoResult{res},
			Errors:  []IDError{},
		}
		return writeJSONEnvelope(env)
	}
	if jsonlOut {
		return writeJSONLines([]infoResult{res}, nil)
	}
	printInfoHuman(res)
	return nil
}

func runInfoBulk(ids []string, cfg config.Config, jsonOut, jsonlOut bool) error {
	var results []infoResult
	var idErrors []IDError
	for _, rawID := range ids {
		id, err := expandProjectID(rawID, cfg)
		if err != nil {
			idErrors = append(idErrors, IDError{
				ID:     rawID,
				Code:   errUnknownID,
				Reason: err.Error(),
			})
			continue
		}
		res, perr := infoProject(id, cfg)
		if perr != nil {
			idErrors = append(idErrors, *perr)
			continue
		}
		results = append(results, res)
	}

	if jsonOut {
		env := Envelope[infoResult]{
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

	for i, r := range results {
		if i > 0 {
			fmt.Println()
		}
		printInfoHuman(r)
	}
	for _, e := range idErrors {
		fmt.Fprintf(os.Stderr, "Error: %s: %s\n", e.ID, e.Reason)
	}
	exitOnPartialFailure(len(idErrors))
	return nil
}

// infoProject builds an infoResult for one resolved ID. Returns either a
// populated infoResult or a non-nil *IDError.
func infoProject(id string, cfg config.Config) (infoResult, *IDError) {
	if cfg.ProjectIDType != "" && !project.IsValidID(id, cfg.ProjectIDType, cfg.ProjectIDPrefix) {
		return infoResult{}, &IDError{
			ID:     id,
			Code:   errInvalidIDFormat,
			Reason: fmt.Sprintf("%q is not a valid project ID (expected %s format)", id, cfg.ProjectIDType),
		}
	}

	projPath := filepath.Join(cfg.ProjectsFolder, id)
	fi, statErr := os.Stat(projPath)
	local := statErr == nil && fi.IsDir()

	hasMetadata := false
	if cfg.MetadataFolder != "" {
		metaDir := cfg.MetadataDir(id)
		if mfi, merr := os.Stat(metaDir); merr == nil && mfi.IsDir() {
			hasMetadata = true
		}
	}

	if !local && !hasMetadata {
		return infoResult{}, &IDError{
			ID:     id,
			Code:   errUnknownID,
			Reason: fmt.Sprintf("project %q not found", id),
		}
	}

	p := resolveProject(id, cfg, local)
	idTime, hasTime := project.ParseIDTime(id)
	isDateOnly := cfg.ProjectIDType == project.FormatAYMDb

	hasReadme := false
	if local {
		if rfi, rerr := os.Stat(filepath.Join(projPath, "README.md")); rerr == nil && !rfi.IsDir() {
			hasReadme = true
		}
	}

	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	res := infoResult{
		ID:        p.ID,
		Title:     p.Title,
		Path:      p.Path,
		Tags:      tags,
		Local:     local,
		HasReadme: hasReadme,
	}
	if hasTime {
		res.Date = idTime.Local().Format("2006-01-02")
		if !isDateOnly {
			res.DateTimeUTC = idTime.Format(time.RFC3339)
		}
	}
	return res, nil
}

func printInfoHuman(r infoResult) {
	fmt.Printf("ID:     %s\n", r.ID)

	if r.Date != "" {
		fmt.Printf("Date:   %s\n", r.Date)
		if r.DateTimeUTC != "" {
			if t, err := time.Parse(time.RFC3339, r.DateTimeUTC); err == nil {
				fmt.Printf("Time:   %s (UTC)\n", t.Format("15:04:05"))
			}
		}
	}

	title := r.Title
	if title == "" {
		title = "(none)"
	}
	fmt.Printf("Title:  %s\n", title)
	fmt.Printf("Path:   %s\n", filepath.Join(r.Path, ""))

	if len(r.Tags) > 0 {
		fmt.Printf("Tags:   %s\n", strings.Join(r.Tags, ", "))
	} else {
		fmt.Printf("Tags:   (none)\n")
	}

	if r.Local {
		fmt.Printf("Local:  yes\n")
	} else {
		fmt.Printf("Local:  no\n")
	}

	if r.Local {
		if r.HasReadme {
			fmt.Printf("README: yes\n")
		} else {
			fmt.Printf("README: no\n")
		}
	}
}

// infoError formats a top-level error for the info command. In non-JSON
// modes, returns a Go error so cobra's RunE printer handles it. In JSON
// mode, emits a top-level error envelope and exits 1. In JSONL mode,
// emits a single error line and exits 1.
func infoError(jsonOut, jsonlOut bool, code, reason string) error {
	if jsonOut {
		env := Envelope[infoResult]{
			Error:   &TopError{Code: code, Reason: reason},
			Results: []infoResult{},
			Errors:  []IDError{},
		}
		_ = writeJSONEnvelope(env)
		os.Exit(1)
		return nil
	}
	if jsonlOut {
		_ = writeJSONLines[infoResult](nil, []IDError{{Code: code, Reason: reason}})
		os.Exit(1)
		return nil
	}
	return fmt.Errorf("%s", reason)
}

