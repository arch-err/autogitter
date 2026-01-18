package main

import (
	"fmt"
	"os"

	"github.com/arch-err/autogitter/internal/config"
	"github.com/arch-err/autogitter/internal/sync"
	"github.com/arch-err/autogitter/internal/ui"
	"github.com/spf13/cobra"
)

var (
	version    = "0.1.0"
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

var (
	syncPrune bool
	syncAdd   bool
	syncForce bool
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to config file (default: $XDG_CONFIG_HOME/autogitter/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")

	syncCmd.Flags().BoolVarP(&syncPrune, "prune", "p", false, "prune repos not in config")
	syncCmd.Flags().BoolVarP(&syncAdd, "add", "a", false, "add orphaned repos to config")
	syncCmd.Flags().BoolVar(&syncForce, "force", false, "skip confirmation prompts")

	rootCmd.AddCommand(syncCmd)
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
	}

	result, err := sync.Run(cfg, opts)
	if err != nil {
		return err
	}

	ui.PrintSummary(result.Cloned, result.Pruned, result.Skipped)

	return nil
}
