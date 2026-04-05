package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/format"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and modify prj configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Print a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	Args:  cobra.NoArgs,
	RunE:  runConfigList,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the configuration file path",
	Args:  cobra.NoArgs,
	RunE:  runConfigPath,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd, configGetCmd, configListCmd, configPathCmd)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	f, ok := config.FieldByKey(key)
	if !ok {
		return fmt.Errorf("unknown config key %q (valid keys: %s)", key, config.ValidKeysHelp())
	}

	// Validate the value via the field's Set (e.g. retention_days must be int).
	var tmp config.Config
	if err := f.Set(&tmp, value); err != nil {
		return err
	}

	path, err := configPath()
	if err != nil {
		return err
	}

	return config.SetField(path, key, value)
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	f, ok := config.FieldByKey(key)
	if !ok {
		return fmt.Errorf("unknown config key %q (valid keys: %s)", key, config.ValidKeysHelp())
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	fmt.Println(f.Get(&cfg))
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	color := format.IsTTY(os.Stdout)

	// Find max key length for alignment.
	maxLen := 0
	for _, f := range config.Fields {
		if len(f.Key) > maxLen {
			maxLen = len(f.Key)
		}
	}

	for _, f := range config.Fields {
		val := f.Get(&cfg)
		switch {
		case cfg.IsExplicit(f.Key):
			fmt.Printf("%-*s = \"%s\"\n", maxLen, f.Key, escapeQuotes(val))
		case !f.IsEmpty(&cfg):
			// Value filled by Load() defaults, not by user.
			valStr := fmt.Sprintf("\"%s\" (default)", escapeQuotes(val))
			if color {
				valStr = "\033[2m" + valStr + "\033[0m"
			}
			fmt.Printf("%-*s = %s\n", maxLen, f.Key, valStr)
		case f.Default != "":
			// Empty in config but has a runtime default applied at point of use.
			valStr := fmt.Sprintf("\"%s\" (default)", escapeQuotes(f.Default))
			if color {
				valStr = "\033[2m" + valStr + "\033[0m"
			}
			fmt.Printf("%-*s = %s\n", maxLen, f.Key, valStr)
		default:
			label := "<not set>"
			if color {
				label = "\033[2m<not set>\033[0m"
			}
			fmt.Printf("%-*s = %s\n", maxLen, f.Key, label)
		}
	}
	return nil
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

// escapeQuotes escapes double quotes for JSON-compatible display.
func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

// configPath returns the effective config file path.
func configPath() (string, error) {
	if cfgFile != "" {
		return cfgFile, nil
	}
	path, err := config.DefaultPath()
	if err != nil {
		return "", fmt.Errorf("resolve config path: %w", err)
	}
	return path, nil
}
