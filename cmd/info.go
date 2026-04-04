package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/project"
)

var infoCmd = &cobra.Command{
	Use:   "info <project-id>",
	Short: "Display information about a project",
	Args:  cobra.ExactArgs(1),
	RunE:  runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().Bool("json", false, "output as JSON")
}

func runInfo(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return infoError(jsonOut, fmt.Errorf("load config: %w", err))
	}

	if err := requireConfig(cfg, "projects_folder"); err != nil {
		return infoError(jsonOut, err)
	}

	id, err := expandProjectID(args[0], cfg)
	if err != nil {
		return infoError(jsonOut, err)
	}

	if cfg.ProjectIDType != "" && !project.IsValidID(id, cfg.ProjectIDType) {
		return infoError(jsonOut, fmt.Errorf("%q is not a valid project ID (expected %s format)", id, cfg.ProjectIDType))
	}

	// Check local folder existence.
	projPath := filepath.Join(cfg.ProjectsFolder, id)
	fi, statErr := os.Stat(projPath)
	local := statErr == nil && fi.IsDir()

	// Check if project is known at all (local folder or metadata).
	hasMetadata := false
	if cfg.MetadataFolder != "" {
		metaDir := cfg.MetadataDir(id)
		if mfi, merr := os.Stat(metaDir); merr == nil && mfi.IsDir() {
			hasMetadata = true
		}
	}

	if !local && !hasMetadata {
		return infoError(jsonOut, fmt.Errorf("project %q not found", id))
	}

	p := resolveProject(id, cfg, local)

	// Parse timestamp from ID.
	idTime, hasTime := project.ParseIDTime(id)
	isDateOnly := cfg.ProjectIDType == project.FormatAYMDb

	// Check README.md presence.
	hasReadme := false
	if local {
		if rfi, rerr := os.Stat(filepath.Join(projPath, "README.md")); rerr == nil && !rfi.IsDir() {
			hasReadme = true
		}
	}

	if jsonOut {
		return printInfoJSON(p, idTime, hasTime, isDateOnly, local, hasReadme)
	}
	printInfoHuman(p, idTime, hasTime, isDateOnly, local, hasReadme)
	return nil
}

func printInfoJSON(p project.Project, idTime time.Time, hasTime, isDateOnly, local, hasReadme bool) error {
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}

	out := map[string]interface{}{
		"id":         p.ID,
		"title":      p.Title,
		"path":       p.Path,
		"tags":       tags,
		"local":      local,
		"has_readme": hasReadme,
	}

	if hasTime {
		localDate := idTime.Local()
		out["date"] = localDate.Format("2006-01-02")
		if !isDateOnly {
			out["datetime_utc"] = idTime.Format(time.RFC3339)
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func printInfoHuman(p project.Project, idTime time.Time, hasTime, isDateOnly, local, hasReadme bool) {
	fmt.Printf("ID:     %s\n", p.ID)

	if hasTime {
		localDate := idTime.Local()
		fmt.Printf("Date:   %s\n", localDate.Format("2006-01-02"))
		if !isDateOnly {
			fmt.Printf("Time:   %s (UTC)\n", idTime.Format("15:04:05"))
		}
	}

	title := p.Title
	if title == "" {
		title = "(none)"
	}
	fmt.Printf("Title:  %s\n", title)
	fmt.Printf("Path:   %s\n", filepath.Join(p.Path, ""))

	if len(p.Tags) > 0 {
		fmt.Printf("Tags:   %s\n", strings.Join(p.Tags, ", "))
	} else {
		fmt.Printf("Tags:   (none)\n")
	}

	if local {
		fmt.Printf("Local:  yes\n")
	} else {
		fmt.Printf("Local:  no\n")
	}

	if local {
		if hasReadme {
			fmt.Printf("README: yes\n")
		} else {
			fmt.Printf("README: no\n")
		}
	}
}

// infoError formats an error for the info command. When jsonOut is true,
// the error is emitted as a JSON object to stdout so scripts can parse it.
func infoError(jsonOut bool, err error) error {
	if !jsonOut {
		return err
	}
	json.NewEncoder(os.Stdout).Encode(map[string]string{"error": err.Error()})
	os.Exit(1)
	return nil // unreachable
}

