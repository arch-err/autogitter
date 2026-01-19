package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arch-err/autogitter/internal/connector"
	"gopkg.in/yaml.v3"
)

type Strategy string

const (
	StrategyManual Strategy = "manual"
	StrategyAll    Strategy = "all"
	StrategyFile   Strategy = "file"
)

type FileStrategy struct {
	Filename string `yaml:"filename"`
}

type Source struct {
	Name         string       `yaml:"name"`
	Source       string       `yaml:"source"`
	Strategy     Strategy     `yaml:"strategy"`
	Type         string       `yaml:"type,omitempty"` // "github", "gitea", or auto-detect from host
	FileStrategy FileStrategy `yaml:"file_strategy,omitempty"`
	LocalPath    string       `yaml:"local_path"`
	PrivateKey   string       `yaml:"private_key,omitempty"`
	Branch       string       `yaml:"branch,omitempty"`
	Repos        []string     `yaml:"repos,omitempty"`
}

type Config struct {
	Sources []Source `yaml:"sources"`
}

func DefaultConfigPath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			configHome = "."
		} else {
			configHome = filepath.Join(home, ".config")
		}
	}
	return filepath.Join(configHome, "autogitter", "config.yaml")
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	cfg.ExpandPaths()

	return &cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Sources) == 0 {
		return fmt.Errorf("no sources defined")
	}

	for i, src := range c.Sources {
		if src.Name == "" {
			return fmt.Errorf("source %d: name is required", i)
		}
		if src.Source == "" {
			return fmt.Errorf("source %q: source URL is required", src.Name)
		}
		if src.LocalPath == "" {
			return fmt.Errorf("source %q: local_path is required", src.Name)
		}

		switch src.Strategy {
		case StrategyManual:
			if len(src.Repos) == 0 {
				return fmt.Errorf("source %q: repos list is required for manual strategy", src.Name)
			}
		case StrategyAll, StrategyFile:
			// Valid strategies for future implementation
		case "":
			return fmt.Errorf("source %q: strategy is required", src.Name)
		default:
			return fmt.Errorf("source %q: unknown strategy %q", src.Name, src.Strategy)
		}

		if src.Strategy == StrategyFile && src.FileStrategy.Filename == "" {
			return fmt.Errorf("source %q: file_strategy.filename is required for file strategy", src.Name)
		}
	}

	return nil
}

func (c *Config) ExpandPaths() {
	for i := range c.Sources {
		c.Sources[i].LocalPath = expandPath(c.Sources[i].LocalPath)
		if c.Sources[i].PrivateKey != "" {
			c.Sources[i].PrivateKey = expandPath(c.Sources[i].PrivateKey)
		}
	}
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	path = os.ExpandEnv(path)

	return path
}

// GetBranch returns the configured branch, or empty string to use remote default
func (s *Source) GetBranch() string {
	return s.Branch
}

func (s *Source) GetRepoURL(repo string) string {
	// Extract just the host from the source (e.g., "github.com" from "github.com/user")
	host := s.GetHost()
	return fmt.Sprintf("git@%s:%s.git", host, repo)
}

// GetHost extracts the host from the source field
func (s *Source) GetHost() string {
	host := s.Source
	if idx := strings.Index(s.Source, "/"); idx != -1 {
		host = s.Source[:idx]
	}
	return host
}

// GetUserOrOrg extracts the user/org from the source field
func (s *Source) GetUserOrOrg() string {
	if idx := strings.Index(s.Source, "/"); idx != -1 {
		return s.Source[idx+1:]
	}
	return ""
}

// GetConnectorType returns the connector type for this source
func (s *Source) GetConnectorType() connector.ConnectorType {
	// If explicitly specified, use that
	if s.Type != "" {
		switch strings.ToLower(s.Type) {
		case "github":
			return connector.ConnectorGitHub
		case "gitea":
			return connector.ConnectorGitea
		}
	}
	// Otherwise, auto-detect from host
	return connector.DetectType(s.GetHost())
}

func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Exists checks if a config file exists at the given path
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DefaultTemplate is the template for new config files
const DefaultTemplate = `# Autogitter Configuration
# See documentation at https://arch-err.github.io/autogitter/configuration

sources:
  # Example source configuration
  - name: "GitHub"
    source: github.com/your-username
    strategy: manual
    local_path: "~/Git/github"
    # branch: main  # optional, uses remote default if not set
    # private_key: "~/.ssh/id_rsa"  # optional, for private repos
    repos:
      - your-username/repo1
      - your-username/repo2
`

// CreateDefault creates a default config file at the given path
func CreateDefault(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(DefaultTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ValidateFile validates a config file without loading it fully
func ValidateFile(path string) error {
	_, err := Load(path)
	return err
}

// ValidateCredentials checks that required API tokens are available for non-manual strategies
func (c *Config) ValidateCredentials() []string {
	var warnings []string

	for _, src := range c.Sources {
		if src.Strategy == StrategyManual {
			continue
		}

		connType := src.GetConnectorType()
		token := connector.GetToken(connType)
		if token == "" {
			envVar := connector.GetEnvVarName(connType)
			warnings = append(warnings, fmt.Sprintf(
				"source %q (strategy: %s) requires %s environment variable",
				src.Name, src.Strategy, envVar,
			))
		}
	}

	return warnings
}
