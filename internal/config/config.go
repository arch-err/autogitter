package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/arch-err/autogitter/internal/connector"
	"gopkg.in/yaml.v3"
)

type Strategy string

const (
	StrategyManual Strategy = "manual"
	StrategyAll    Strategy = "all"
	StrategyFile   Strategy = "file"
	StrategyRegex  Strategy = "regex"
)

type FileStrategy struct {
	Filename string `yaml:"filename"`
}

type RegexStrategy struct {
	Pattern string `yaml:"pattern"`
}

// RepoEntry represents a repository in the config.
// It supports both plain string format ("user/repo") and object format
// with an optional local_path override.
type RepoEntry struct {
	Name      string `yaml:"name"`
	LocalPath string `yaml:"local_path,omitempty"`
}

// UnmarshalYAML allows RepoEntry to be unmarshaled from either a plain string
// or a mapping with name and local_path fields.
func (r *RepoEntry) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		r.Name = value.Value
		return nil
	}
	if value.Kind == yaml.MappingNode {
		// Use an alias type to avoid infinite recursion
		type repoEntryRaw struct {
			Name      string `yaml:"name"`
			LocalPath string `yaml:"local_path,omitempty"`
		}
		var raw repoEntryRaw
		if err := value.Decode(&raw); err != nil {
			return err
		}
		r.Name = raw.Name
		r.LocalPath = raw.LocalPath
		return nil
	}
	return fmt.Errorf("expected string or mapping for repo entry, got %v", value.Kind)
}

// MarshalYAML emits a plain string when no LocalPath is set, or an object otherwise.
func (r RepoEntry) MarshalYAML() (interface{}, error) {
	if r.LocalPath == "" {
		return r.Name, nil
	}
	return struct {
		Name      string `yaml:"name"`
		LocalPath string `yaml:"local_path"`
	}{
		Name:      r.Name,
		LocalPath: r.LocalPath,
	}, nil
}

// ResolvedLocalPath returns the effective local path for this repo.
// If LocalPath is set, it returns that; otherwise filepath.Join(sourceLocalPath, baseName).
func (r RepoEntry) ResolvedLocalPath(sourceLocalPath string) string {
	if r.LocalPath != "" {
		return r.LocalPath
	}
	return filepath.Join(sourceLocalPath, repoBaseName(r.Name))
}

// HasCustomLocalPath returns true if this repo has a custom local_path override.
func (r RepoEntry) HasCustomLocalPath() bool {
	return r.LocalPath != ""
}

// RepoEntriesFromNames converts a slice of repo name strings to a RepoEntry slice.
func RepoEntriesFromNames(names []string) []RepoEntry {
	entries := make([]RepoEntry, len(names))
	for i, name := range names {
		entries[i] = RepoEntry{Name: name}
	}
	return entries
}

// repoBaseName extracts the repo name from a full "user/repo" string.
func repoBaseName(fullName string) string {
	parts := strings.Split(fullName, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
}

type SSHOptions struct {
	Port       int    `yaml:"port,omitempty"`
	PrivateKey string `yaml:"private_key,omitempty"`
}

type Source struct {
	Name          string        `yaml:"name"`
	Source        string        `yaml:"source"`
	Strategy      Strategy      `yaml:"strategy"`
	Type          string        `yaml:"type,omitempty"` // "github", "gitea", "bitbucket", or auto-detect from host
	FileStrategy  FileStrategy  `yaml:"file_strategy,omitempty"`
	RegexStrategy RegexStrategy `yaml:"regex_strategy,omitempty"`
	LocalPath     string        `yaml:"local_path"`
	SSHOptions    SSHOptions    `yaml:"ssh_options,omitempty"`
	PrivateKey    string        `yaml:"private_key,omitempty"` // deprecated: use ssh_options.private_key
	Branch        string        `yaml:"branch,omitempty"`
	Repos         []RepoEntry   `yaml:"repos,omitempty"`
}

type Config struct {
	Sources []Source `yaml:"sources"`
}

func DefaultConfigPath() string {
	return filepath.Join(configDir(), "config.yaml")
}

// SourcesDirPath returns the path to the sources.d directory
func SourcesDirPath() string {
	return filepath.Join(configDir(), "sources.d")
}

func configDir() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			configHome = "."
		} else {
			configHome = filepath.Join(home, ".config")
		}
	}
	return filepath.Join(configHome, "autogitter")
}

func Load(path string) (*Config, error) {
	var cfg Config

	data, err := readConfig(path)
	if err != nil {
		// For local configs, allow missing config file if sources.d provides sources
		if !IsRemote(path) && os.IsNotExist(err) {
			// Continue with empty config, sources.d may provide sources
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Load additional sources from sources.d directory (only for local configs)
	if !IsRemote(path) {
		sourcesDir := filepath.Join(filepath.Dir(path), "sources.d")
		if err := cfg.loadSourcesDir(sourcesDir); err != nil {
			return nil, fmt.Errorf("failed to load sources.d: %w", err)
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	cfg.ExpandPaths()

	return &cfg, nil
}

// loadSourcesDir loads all yaml files from the sources.d directory and merges them
func (c *Config) loadSourcesDir(dir string) error {
	// Check if directory exists
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil // sources.d is optional
	}
	if err != nil {
		return fmt.Errorf("failed to stat sources.d: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("sources.d is not a directory")
	}

	// Read directory entries
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read sources.d: %w", err)
	}

	// Process files in alphabetical order (ReadDir already sorts)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Only process .yaml and .yml files
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		filePath := filepath.Join(dir, name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", name, err)
		}

		var fileCfg Config
		if err := yaml.Unmarshal(data, &fileCfg); err != nil {
			return fmt.Errorf("failed to parse %s: %w", name, err)
		}

		// Append sources from this file
		c.Sources = append(c.Sources, fileCfg.Sources...)
	}

	return nil
}

// readConfig reads config data from a local file, HTTP URL, or SSH path
func readConfig(path string) ([]byte, error) {
	// HTTP/HTTPS URL
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return fetchHTTP(path)
	}

	// SSH URL (ssh://user@host/path or user@host:/path)
	if strings.HasPrefix(path, "ssh://") || isSSHPath(path) {
		return fetchSSH(path)
	}

	// Local file
	return os.ReadFile(path)
}

// isSSHPath checks if path looks like an SSH path (user@host:/path)
func isSSHPath(path string) bool {
	// Must contain @ and : but not be a Windows path (C:\)
	atIdx := strings.Index(path, "@")
	colonIdx := strings.Index(path, ":")
	return atIdx > 0 && colonIdx > atIdx && !strings.HasPrefix(path[colonIdx:], ":\\")
}

// fetchHTTP fetches config from an HTTP/HTTPS URL
func fetchHTTP(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch config: HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// fetchSSH fetches config from a remote host via SSH
func fetchSSH(path string) ([]byte, error) {
	var host, remotePath string

	if strings.HasPrefix(path, "ssh://") {
		// ssh://user@host/path/to/config.yaml
		path = strings.TrimPrefix(path, "ssh://")
		// Find the first / after the host
		slashIdx := strings.Index(path, "/")
		if slashIdx == -1 {
			return nil, fmt.Errorf("invalid SSH URL: missing path")
		}
		host = path[:slashIdx]
		remotePath = path[slashIdx:]
	} else {
		// user@host:/path/to/config.yaml
		colonIdx := strings.Index(path, ":")
		host = path[:colonIdx]
		remotePath = path[colonIdx+1:]
	}

	// Use ssh to cat the remote file
	cmd := exec.Command("ssh", host, "cat", remotePath)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("SSH failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("SSH failed: %w", err)
	}

	return output, nil
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
			for _, repo := range src.Repos {
				if repo.Name == "" {
					return fmt.Errorf("source %q: repo name is required", src.Name)
				}
			}
		case StrategyAll, StrategyFile, StrategyRegex:
			// Valid strategies that fetch from API
		case "":
			return fmt.Errorf("source %q: strategy is required", src.Name)
		default:
			return fmt.Errorf("source %q: unknown strategy %q", src.Name, src.Strategy)
		}

		if src.Strategy == StrategyFile && src.FileStrategy.Filename == "" {
			return fmt.Errorf("source %q: file_strategy.filename is required for file strategy", src.Name)
		}

		if src.Strategy == StrategyRegex {
			if src.RegexStrategy.Pattern == "" {
				return fmt.Errorf("source %q: regex_strategy.pattern is required for regex strategy", src.Name)
			}
			if _, err := regexp.Compile(src.RegexStrategy.Pattern); err != nil {
				return fmt.Errorf("source %q: invalid regex pattern: %w", src.Name, err)
			}
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
		if c.Sources[i].SSHOptions.PrivateKey != "" {
			c.Sources[i].SSHOptions.PrivateKey = expandPath(c.Sources[i].SSHOptions.PrivateKey)
		}
		for j := range c.Sources[i].Repos {
			if c.Sources[i].Repos[j].LocalPath != "" {
				c.Sources[i].Repos[j].LocalPath = expandPath(c.Sources[i].Repos[j].LocalPath)
			}
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
	host := s.GetHost()

	// If custom SSH port is specified, use ssh:// URL format
	if s.SSHOptions.Port > 0 {
		return fmt.Sprintf("ssh://git@%s:%d/%s.git", host, s.SSHOptions.Port, repo)
	}

	// Standard git@ URL format
	return fmt.Sprintf("git@%s:%s.git", host, repo)
}

// GetPrivateKey returns the SSH private key path, checking both locations
func (s *Source) GetPrivateKey() string {
	// Prefer ssh_options.private_key over deprecated top-level private_key
	if s.SSHOptions.PrivateKey != "" {
		return s.SSHOptions.PrivateKey
	}
	return s.PrivateKey
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
		case "bitbucket":
			return connector.ConnectorBitbucket
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
// For remote URLs (HTTP/SSH), always returns true (validation will fail if not accessible)
func Exists(path string) bool {
	if IsRemote(path) {
		return true
	}
	_, err := os.Stat(path)
	return err == nil
}

// IsRemote checks if the path is a remote URL (HTTP or SSH)
func IsRemote(path string) bool {
	return strings.HasPrefix(path, "http://") ||
		strings.HasPrefix(path, "https://") ||
		strings.HasPrefix(path, "ssh://") ||
		isSSHPath(path)
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
    # ssh_options:
    #   port: 22  # optional, for non-standard SSH port
    #   private_key: "~/.ssh/id_rsa"  # optional, for private repos
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
