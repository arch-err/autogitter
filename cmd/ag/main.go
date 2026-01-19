package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/arch-err/autogitter/internal/config"
	"github.com/arch-err/autogitter/internal/sync"
	"github.com/arch-err/autogitter/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var (
	version    = "0.2.0"
	configPath string
	debug      bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "ag",
	Short:   "Autogitter - Git repository synchronization tool",
	Long:    `Autogitter (ag) is a tool to synchronize git repositories based on a configuration file.`,
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ui.SetDebug(debug)
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync repositories according to config",
	Long:  `Sync clones missing repositories and optionally prunes or adds orphaned ones.`,
	RunE:  runSync,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Edit configuration file",
	Long:  `Opens the configuration file in your default editor. Creates a template if the file doesn't exist.`,
	RunE:  runConfig,
}

var (
	syncPrune      bool
	syncAdd        bool
	syncForce      bool
	syncJobs       int
	configValidate bool
	configGenerate bool
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to config file (default: $XDG_CONFIG_HOME/autogitter/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")

	syncCmd.Flags().BoolVarP(&syncPrune, "prune", "p", false, "prune repos not in config")
	syncCmd.Flags().BoolVarP(&syncAdd, "add", "a", false, "add orphaned repos to config")
	syncCmd.Flags().BoolVar(&syncForce, "force", false, "skip confirmation prompts")
	syncCmd.Flags().IntVarP(&syncJobs, "jobs", "j", 4, "number of parallel clone workers")

	rootCmd.AddCommand(syncCmd)

	configCmd.Flags().BoolVarP(&configValidate, "validate", "v", false, "validate config file without editing")
	configCmd.Flags().BoolVarP(&configGenerate, "generate", "g", false, "generate default config file")
	rootCmd.AddCommand(configCmd)
}

func loadConfig() (*config.Config, string, error) {
	path := configPath
	if path == "" {
		path = config.DefaultConfigPath()
	}

	cfg, err := config.Load(path)
	if err != nil {
		return nil, path, err
	}

	return cfg, path, nil
}

func runSync(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		ui.Error("failed to load config", "error", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	ui.Info("loaded config", "path", cfgPath, "sources", len(cfg.Sources))

	opts := sync.SyncOptions{
		Prune:      syncPrune,
		Add:        syncAdd,
		Force:      syncForce,
		ConfigPath: cfgPath,
		Jobs:       syncJobs,
	}

	result, err := sync.Run(cfg, opts)
	if err != nil {
		return err
	}

	ui.PrintSummary(result.Cloned, result.Pruned, result.Skipped)

	return nil
}

func runConfig(cmd *cobra.Command, args []string) error {
	path := configPath
	if path == "" {
		path = config.DefaultConfigPath()
	}

	// Generate only mode
	if configGenerate {
		if config.Exists(path) {
			return fmt.Errorf("config file already exists: %s", path)
		}
		if err := config.CreateDefault(path); err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}
		ui.Info("config created", "path", path)
		return nil
	}

	// Validate only mode
	if configValidate {
		if !config.Exists(path) {
			return fmt.Errorf("config file not found: %s", path)
		}
		if err := config.ValidateFile(path); err != nil {
			ui.Error("config validation failed", "error", err)
			return err
		}
		ui.Info("config is valid", "path", path)
		return nil
	}

	// Create default config if it doesn't exist
	if !config.Exists(path) {
		ui.Info("creating default config", "path", path)
		if err := config.CreateDefault(path); err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}
	}

	// Open in editor
	editor := getEditor()
	ui.Info("opening config in editor", "editor", editor, "path", path)

	for {
		// Run editor
		editorCmd := exec.Command(editor, path)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("failed to run editor: %w", err)
		}

		// Validate config after editing
		if err := config.ValidateFile(path); err != nil {
			ui.Error("config validation failed", "error", err)

			var retry bool
			huh.NewConfirm().
				Title("Config validation failed. Edit again?").
				Affirmative("Yes, edit again").
				Negative("No, discard changes").
				Value(&retry).
				Run()

			if retry {
				continue
			}
			return fmt.Errorf("config validation failed: %w", err)
		}

		ui.Info("config saved successfully", "path", path)
		break
	}

	return nil
}

func getEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	// Try common editors
	for _, editor := range []string{"vim", "nano", "vi"} {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}
	return "vi" // fallback
}
