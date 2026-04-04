package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set via ldflags at build time.
var version = "dev"

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "prj",
	Short: "Project folder manager",
	Long:  "prj manages project folders, their metadata, and cross-platform links.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $UserConfigDir/prj/config.json)")
	rootCmd.Version = version
}
