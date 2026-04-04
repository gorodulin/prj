package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/project"
)

var pathCmd = &cobra.Command{
	Use:   "path <project-id>",
	Short: "Print the full path to a project folder",
	Args:  cobra.ExactArgs(1),
	RunE:  runPath,
}

func init() {
	rootCmd.AddCommand(pathCmd)
	pathCmd.Flags().Bool("strict", false, "error if project folder does not exist locally")
}

func runPath(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := requireConfig(cfg, "projects_folder"); err != nil {
		return err
	}

	id, err := expandProjectID(args[0], cfg)
	if err != nil {
		return err
	}

	if cfg.ProjectIDType != "" && !project.IsValidID(id, cfg.ProjectIDType) {
		return fmt.Errorf("%q is not a valid project ID (expected %s format)", id, cfg.ProjectIDType)
	}

	strict, _ := cmd.Flags().GetBool("strict")
	projPath := filepath.Join(cfg.ProjectsFolder, id)

	info, err := os.Stat(projPath)
	exists := err == nil && info.IsDir()

	if strict && !exists {
		return fmt.Errorf("project folder does not exist: %s", projPath)
	}

	if !exists {
		fmt.Fprintln(os.Stderr, "Warning: project folder does not exist locally")
	}

	fmt.Println(projPath)
	return nil
}
