package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/metadata"
	"github.com/gorodulin/prj/internal/project"
	"github.com/gorodulin/prj/internal/text"
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new project",
	RunE:  runNew,
}

func init() {
	rootCmd.AddCommand(newCmd)
	newCmd.Flags().String("title", "", "project title")
	newCmd.Flags().String("tags", "", "comma-separated tags")
	newCmd.Flags().String("tag", "", "deprecated alias for --tags")
	_ = newCmd.Flags().MarkDeprecated("tag", "use --tags instead")
	newCmd.Flags().Bool("readme", false, "create README.md with front matter")
	newCmd.MarkFlagsMutuallyExclusive("tags", "tag")
}

func runNew(cmd *cobra.Command, args []string) (err error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	title, _ := cmd.Flags().GetString("title")
	tagsRaw := flagString(cmd, "tags")
	if !cmd.Flags().Changed("tags") {
		tagsRaw = flagString(cmd, "tag")
	}
	tags := text.ParseTags(tagsRaw)
	createReadme, _ := cmd.Flags().GetBool("readme")
	hasTitle := title != ""
	writeMeta := hasTitle || len(tags) > 0

	required := []string{"projects_folder"}
	if writeMeta {
		required = append(required, metadataWriteKeys...)
	}
	if err := requireConfig(cfg, required...); err != nil {
		return err
	}

	ids, err := collectIDs(cfg)
	if err != nil {
		return err
	}

	newID, err := project.GenerateID(cfg.ProjectIDType, ids.all, cfg.ProjectIDPrefix)
	if err != nil {
		return fmt.Errorf("generate project ID: %w", err)
	}

	projPath := filepath.Join(cfg.ProjectsFolder, newID)
	if err := os.Mkdir(projPath, 0755); err != nil {
		return fmt.Errorf("create project folder %s: %w", projPath, err)
	}
	metaDir := cfg.MetadataDir(newID)
	defer func() {
		if err != nil {
			_ = os.RemoveAll(projPath)
			_ = os.Remove(metaDir) // only succeeds if left empty by a failed snapshot write
		}
	}()

	// README goes before the snapshot so a failure only leaves state inside projPath.
	if createReadme {
		if !hasTitle {
			fmt.Fprintln(os.Stderr, "Warning: --readme requires --title, skipping README creation")
		} else {
			content := project.BuildReadme(title, tags)
			readmePath := filepath.Join(projPath, "README.md")
			if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
				return fmt.Errorf("write README.md: %w", err)
			}
		}
	}

	if writeMeta {
		var titleSet *string
		if hasTitle {
			titleSet = &title
		}
		s := metadata.Snapshot{
			TitleSet:    titleSet,
			Tags:        tags,
			TagsAdded:   tags,
			MachineID:   cfg.MachineID,
			MachineName: cfg.MachineName,
			Version:     1,
		}
		if _, err := metadata.WriteSnapshot(metaDir, s); err != nil {
			return fmt.Errorf("write metadata for %s: %w", newID, err)
		}
		if n, err := metadata.PurgeOldSnapshots(metaDir, cfg.RetentionDays); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: purge old snapshots for %s: %v\n", newID, err)
		} else if n > 0 {
			fmt.Fprintf(os.Stderr, "Purged %d old snapshot(s) for %s\n", n, newID)
		}
	}

	fmt.Printf("%s\t%s\n", newID, projPath)
	return nil
}

func flagString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}
