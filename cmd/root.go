package cmd

import (
	"fmt"
	"os"
	"slices"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/format"
)

// version is set via ldflags at build time.
var version = "dev"

var (
	cfgFile   string
	noColor   bool
	colorMode string
)

var rootCmd = &cobra.Command{
	Use:   "prj",
	Short: "Projector — manage project folders, metadata, and links",
	Long:  "Projector (prj) manages project folders, their metadata, and cross-platform links.",
	// Cobra runs only the NEAREST PersistentPreRunE in the command chain.
	// If a subcommand ever defines its own, it must call this validation
	// (or cobra.EnableTraverseRunHooks must be revisited).
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("color") && !slices.Contains(format.ValidColorModes, colorMode) {
			return fmt.Errorf("--color %q is not recognized (use %s)",
				colorMode, config.JoinQuoted(format.ValidColorModes))
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $UserConfigDir/prj/config.json)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().StringVar(&colorMode, "color", "", "color output mode: auto, always, or never (overrides config)")
	rootCmd.Version = version
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
}
