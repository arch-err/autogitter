package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func (s *Source) GetBranch() string {
	if s.Branch != "" {
		return s.Branch
	}
	return "main"
}

func (s *Source) GetRepoURL(repo string) string {
	// Extract just the host from the source (e.g., "github.com" from "github.com/user")
	host := s.Source
	if idx := strings.Index(s.Source, "/"); idx != -1 {
		host = s.Source[:idx]
	}
	return fmt.Sprintf("git@%s:%s.git", host, repo)
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
