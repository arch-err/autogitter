package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/arch-err/autogitter/internal/config"
	"github.com/arch-err/autogitter/internal/connector"
	"github.com/arch-err/autogitter/internal/sync"
	"github.com/arch-err/autogitter/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var (
	version    = "0.7.0"
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

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Configure API authentication",
	Long:  `Set up API tokens for GitHub, Gitea, or other Git providers to enable the "all" and "file" sync strategies.`,
	RunE:  runConnect,
}

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull updates for all local repos",
	Long:  `Pull runs git pull on all repositories found in the configured source directories.`,
	RunE:  runPull,
}

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show diff between local repos and config",
	Long:  `Shows a unified diff-style output comparing local repository state against the configuration.`,
	RunE:  runDiff,
}

var (
	syncPrune      bool
	syncAdd        bool
	syncForce      bool
	syncJobs       int
	syncDryRun     bool
	pullForce      bool
	pullJobs       int
	configValidate bool
	configGenerate bool
	connectType    string
	connectHost    string
	connectToken   string
	connectList    bool
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to config file (default: $XDG_CONFIG_HOME/autogitter/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")

	syncCmd.Flags().BoolVarP(&syncPrune, "prune", "p", false, "prune repos not in config")
	syncCmd.Flags().BoolVarP(&syncAdd, "add", "a", false, "add orphaned repos to config")
	syncCmd.Flags().BoolVar(&syncForce, "force", false, "skip confirmation prompts")
	syncCmd.Flags().IntVarP(&syncJobs, "jobs", "j", 4, "number of parallel clone workers")
	syncCmd.Flags().BoolVarP(&syncDryRun, "dry-run", "n", false, "show what would happen without making changes")

	rootCmd.AddCommand(syncCmd)

	pullCmd.Flags().BoolVar(&pullForce, "force", false, "skip confirmation prompts")
	pullCmd.Flags().IntVarP(&pullJobs, "jobs", "j", 4, "number of parallel pull workers")
	rootCmd.AddCommand(pullCmd)

	rootCmd.AddCommand(diffCmd)

	configCmd.Flags().BoolVarP(&configValidate, "validate", "v", false, "validate config file without editing")
	configCmd.Flags().BoolVarP(&configGenerate, "generate", "g", false, "generate default config file")
	rootCmd.AddCommand(configCmd)

	connectCmd.Flags().StringVarP(&connectType, "type", "t", "", "connector type (github|gitea|bitbucket)")
	connectCmd.Flags().StringVarP(&connectHost, "host", "H", "", "git server host (e.g., gitea.company.com)")
	connectCmd.Flags().StringVarP(&connectToken, "token", "T", "", "API token (skips interactive prompt)")
	connectCmd.Flags().BoolVarP(&connectList, "list", "l", false, "list configured connections")
	rootCmd.AddCommand(connectCmd)
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
		DryRun:     syncDryRun,
	}

	result, err := sync.Run(cfg, opts)
	if err != nil {
		return err
	}

	ui.PrintSummary(result.Cloned, result.Pruned, result.Skipped)

	return nil
}

func runPull(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		ui.Error("failed to load config", "error", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	ui.Info("loaded config", "path", cfgPath, "sources", len(cfg.Sources))

	opts := sync.PullOptions{
		Force: pullForce,
		Jobs:  pullJobs,
	}

	result, err := sync.RunPull(cfg, opts)
	if err != nil {
		return err
	}

	ui.Info("pull complete", "updated", result.Updated, "failed", result.Failed)

	return nil
}

func runDiff(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		ui.Error("failed to load config", "error", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	ui.Debug("loaded config", "path", cfgPath, "sources", len(cfg.Sources))

	var diffs []ui.SourceDiff

	for i := range cfg.Sources {
		source := &cfg.Sources[i]

		statuses, err := sync.ComputeSourceStatus(source)
		if err != nil {
			ui.Warn("skipping source", "source", source.Name, "error", err)
			continue
		}

		// Convert RepoStatus to DiffEntry
		entries := make([]ui.DiffEntry, len(statuses))
		for j, s := range statuses {
			entries[j] = ui.DiffEntry{Name: s.Name, Status: s.Status}
		}

		diffs = append(diffs, ui.SourceDiff{
			Name:    source.Name,
			Entries: entries,
		})
	}

	if len(diffs) == 0 {
		ui.Info("no sources to diff")
		return nil
	}

	ui.PrintUnifiedDiff(diffs)

	return nil
}

func runConfig(cmd *cobra.Command, args []string) error {
	path := configPath
	if path == "" {
		path = config.DefaultConfigPath()
	}

	// Generate only mode - output template to stdout
	if configGenerate {
		fmt.Print(config.DefaultTemplate)
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

	// Cannot edit remote configs
	if config.IsRemote(path) {
		return fmt.Errorf("cannot edit remote config, use --validate to check it")
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
			if err := huh.NewConfirm().
				Title("Config validation failed. Edit again?").
				Affirmative("Yes, edit again").
				Negative("No, discard changes").
				Value(&retry).
				Run(); err != nil {
				return fmt.Errorf("prompt failed: %w", err)
			}

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

func runConnect(cmd *cobra.Command, args []string) error {
	// Load existing credentials (ignore error - credentials may not exist yet)
	credPath := connector.DefaultCredentialsPath()
	_ = connector.LoadCredentialsEnv(credPath)

	// List mode
	if connectList {
		return listConnections()
	}

	var connType connector.ConnectorType
	var host string
	var token string

	// Non-interactive mode if type and token are provided
	if connectType != "" && connectToken != "" {
		switch connectType {
		case "github":
			connType = connector.ConnectorGitHub
			host = "github.com"
			if connectHost != "" {
				host = strings.TrimPrefix(connectHost, "https://")
				host = strings.TrimPrefix(host, "http://")
				host = strings.TrimSuffix(host, "/")
			}
		case "gitea":
			connType = connector.ConnectorGitea
			if connectHost == "" {
				host = "gitea.com"
			} else {
				host = strings.TrimPrefix(connectHost, "https://")
				host = strings.TrimPrefix(host, "http://")
				host = strings.TrimSuffix(host, "/")
			}
		case "bitbucket":
			connType = connector.ConnectorBitbucket
			if connectHost == "" {
				host = "bitbucket.org"
			} else {
				host = strings.TrimPrefix(connectHost, "https://")
				host = strings.TrimPrefix(host, "http://")
				host = strings.TrimSuffix(host, "/")
			}
		default:
			return fmt.Errorf("unknown connector type: %s", connectType)
		}
		token = connectToken
	} else {
		// Interactive mode
		var err error
		connType, host, token, err = interactiveConnect()
		if err != nil {
			return err
		}
	}

	// Test the connection
	ui.Info("testing connection...")
	conn, err := connector.New(connType, host, token)
	if err != nil {
		return fmt.Errorf("failed to create connector: %w", err)
	}

	ctx := context.Background()
	if err := conn.TestConnection(ctx); err != nil {
		ui.Error("connection test failed", "error", err)
		return fmt.Errorf("connection test failed: %w", err)
	}

	ui.Info("connection successful")

	// Save the credential
	envVar := connector.GetEnvVarName(connType)
	if err := connector.SaveCredential(credPath, envVar, token); err != nil {
		return fmt.Errorf("failed to save credential: %w", err)
	}

	ui.Info("credential saved", "path", credPath)
	fmt.Println()
	fmt.Printf("To use in the current session, run:\n")
	fmt.Printf("  export %s=%s\n", envVar, token)
	fmt.Println()

	return nil
}

func interactiveConnect() (connector.ConnectorType, string, string, error) {
	// Select connector type
	var typeChoice string
	err := huh.NewSelect[string]().
		Title("Select Git provider").
		Options(
			huh.NewOption("GitHub (github.com)", "github"),
			huh.NewOption("Gitea (gitea.com)", "gitea"),
			huh.NewOption("Bitbucket (bitbucket.org)", "bitbucket"),
			huh.NewOption("Custom (self-hosted)", "custom"),
		).
		Value(&typeChoice).
		Run()

	if err != nil {
		return "", "", "", err
	}

	var connType connector.ConnectorType
	var host string
	var tokenURL string

	switch typeChoice {
	case "github":
		connType = connector.ConnectorGitHub
		host = "github.com"
		tokenURL = "https://github.com/settings/tokens/new?description=autogitter&scopes=repo"
	case "gitea":
		connType = connector.ConnectorGitea
		host = "gitea.com"
		tokenURL = "https://gitea.com/user/settings/applications"
	case "bitbucket":
		connType = connector.ConnectorBitbucket
		host = "bitbucket.org"
		tokenURL = "https://bitbucket.org/account/settings/app-passwords/"
	case "custom":
		// Ask for host
		err := huh.NewInput().
			Title("Enter server host").
			Placeholder("git.example.com").
			Value(&host).
			Run()

		if err != nil {
			return "", "", "", err
		}

		if host == "" {
			return "", "", "", fmt.Errorf("host is required")
		}

		// Strip protocol prefix if present
		host = strings.TrimPrefix(host, "https://")
		host = strings.TrimPrefix(host, "http://")
		host = strings.TrimSuffix(host, "/")

		// Ask for provider type
		var providerType string
		err = huh.NewSelect[string]().
			Title("Select provider type").
			Options(
				huh.NewOption("GitHub Enterprise", "github"),
				huh.NewOption("Gitea", "gitea"),
				huh.NewOption("Bitbucket Server", "bitbucket"),
			).
			Value(&providerType).
			Run()

		if err != nil {
			return "", "", "", err
		}

		switch providerType {
		case "github":
			connType = connector.ConnectorGitHub
			tokenURL = fmt.Sprintf("https://%s/settings/tokens/new", host)
		case "gitea":
			connType = connector.ConnectorGitea
			tokenURL = fmt.Sprintf("https://%s/user/settings/applications", host)
		case "bitbucket":
			connType = connector.ConnectorBitbucket
			tokenURL = fmt.Sprintf("https://%s/account", host)
		}
	}

	// Show token generation instructions
	fmt.Println()
	fmt.Printf("Generate an access token at:\n")
	fmt.Printf("  %s\n", tokenURL)
	if connType == connector.ConnectorBitbucket && host != "bitbucket.org" {
		fmt.Printf("  (click 'HTTP access tokens' in the menu)\n")
	}
	// Copy URL to clipboard
	ui.CopyToClipboard(tokenURL)
	fmt.Printf("  \033[90m📋 copied to clipboard\033[0m\n")
	fmt.Println()
	fmt.Printf("Required permissions:\n")
	switch connType {
	case connector.ConnectorGitHub:
		fmt.Printf("  - repo (Full control of private repositories)\n")
	case connector.ConnectorBitbucket:
		fmt.Printf("  - Repository: Read\n")
	default:
		fmt.Printf("  - read:user (to verify authentication)\n")
		fmt.Printf("  - read:repository (to list repositories)\n")
	}
	fmt.Println()

	// Prompt for token
	var token string
	err = huh.NewInput().
		Title("Enter your access token").
		EchoMode(huh.EchoModePassword).
		Value(&token).
		Run()

	if err != nil {
		return "", "", "", err
	}

	if token == "" {
		return "", "", "", fmt.Errorf("token is required")
	}

	return connType, host, token, nil
}

func listConnections() error {
	credPath := connector.DefaultCredentialsPath()

	fmt.Println("Configured connections:")
	fmt.Println()

	hasAny := false

	// Check GitHub
	if token := connector.GetToken(connector.ConnectorGitHub); token != "" {
		masked := maskToken(token)
		fmt.Printf("  GitHub:    %s\n", masked)
		hasAny = true
	}

	// Check Gitea
	if token := connector.GetToken(connector.ConnectorGitea); token != "" {
		masked := maskToken(token)
		fmt.Printf("  Gitea:     %s\n", masked)
		hasAny = true
	}

	// Check Bitbucket
	if token := connector.GetToken(connector.ConnectorBitbucket); token != "" {
		masked := maskToken(token)
		fmt.Printf("  Bitbucket: %s\n", masked)
		hasAny = true
	}

	if !hasAny {
		fmt.Println("  No connections configured.")
		fmt.Println()
		fmt.Println("Run 'ag connect' to set up a connection.")
	}

	fmt.Println()
	fmt.Printf("Credentials file: %s\n", credPath)

	return nil
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
