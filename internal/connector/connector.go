package connector

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Connector interface for Git providers
type Connector interface {
	// ListRepos returns all repos for the configured user/org
	ListRepos(ctx context.Context, userOrOrg string) ([]string, error)
	// TestConnection verifies the token works
	TestConnection(ctx context.Context) error
	// Name returns the connector type name
	Name() string
}

// ConnectorType represents the type of Git provider
type ConnectorType string

const (
	ConnectorGitHub    ConnectorType = "github"
	ConnectorGitea     ConnectorType = "gitea"
	ConnectorBitbucket ConnectorType = "bitbucket"
)

// New creates a new connector based on type
func New(connType ConnectorType, host string, token string) (Connector, error) {
	switch connType {
	case ConnectorGitHub:
		return NewGitHubConnector(host, token), nil
	case ConnectorGitea:
		return NewGiteaConnector(host, token), nil
	case ConnectorBitbucket:
		return NewBitbucketConnector(host, token), nil
	default:
		return nil, fmt.Errorf("unknown connector type: %s", connType)
	}
}

// DetectType auto-detects connector type from host
func DetectType(host string) ConnectorType {
	host = strings.ToLower(host)
	if strings.Contains(host, "github.com") {
		return ConnectorGitHub
	}
	if strings.Contains(host, "bitbucket.org") {
		return ConnectorBitbucket
	}
	if strings.Contains(host, "gitea.com") {
		return ConnectorGitea
	}
	// Default to Gitea for self-hosted instances
	return ConnectorGitea
}

// DefaultCredentialsPath returns the default path for credentials.env
func DefaultCredentialsPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "autogitter", "credentials.env")
}

// LoadCredentialsEnv loads environment variables from a credentials file
func LoadCredentialsEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Not an error if file doesn't exist
		}
		return fmt.Errorf("failed to open credentials file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		value = strings.Trim(value, `"'`)

		// Only set if not already set in environment
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

// SaveCredential saves or updates a credential in the credentials file
func SaveCredential(path, key, value string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	// Read existing content
	existingLines := make(map[string]bool)
	var lines []string

	file, err := os.Open(path)
	if err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)

			// Check if this line sets the key we're updating
			if !strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, "=") {
				parts := strings.SplitN(trimmed, "=", 2)
				if strings.TrimSpace(parts[0]) == key {
					// Replace this line
					lines = append(lines, fmt.Sprintf("%s=%s", key, value))
					existingLines[key] = true
					continue
				}
			}
			lines = append(lines, line)
		}
		file.Close()
	}

	// Add new key if it wasn't found
	if !existingLines[key] {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	// Write back
	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return os.WriteFile(path, []byte(content), 0600)
}

// GetToken retrieves the token for a connector type
func GetToken(connType ConnectorType) string {
	switch connType {
	case ConnectorGitHub:
		// Check GITHUB_TOKEN first
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			return token
		}
		// Fall back to gh CLI token
		return getGhCliToken("github.com")
	case ConnectorGitea:
		return os.Getenv("GITEA_TOKEN")
	case ConnectorBitbucket:
		return os.Getenv("BITBUCKET_TOKEN")
	default:
		return ""
	}
}

// ghHostsConfig represents the gh CLI hosts.yml structure
type ghHostsConfig map[string]struct {
	OAuthToken string `yaml:"oauth_token"`
	User       string `yaml:"user"`
}

// getGhCliToken reads the token from gh CLI config
func getGhCliToken(host string) string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configHome = filepath.Join(home, ".config")
	}

	hostsPath := filepath.Join(configHome, "gh", "hosts.yml")
	data, err := os.ReadFile(hostsPath)
	if err != nil {
		return ""
	}

	var hosts ghHostsConfig
	if err := yaml.Unmarshal(data, &hosts); err != nil {
		return ""
	}

	if hostConfig, ok := hosts[host]; ok {
		return hostConfig.OAuthToken
	}

	return ""
}

// GetEnvVarName returns the environment variable name for a connector type
func GetEnvVarName(connType ConnectorType) string {
	switch connType {
	case ConnectorGitHub:
		return "GITHUB_TOKEN"
	case ConnectorGitea:
		return "GITEA_TOKEN"
	case ConnectorBitbucket:
		return "BITBUCKET_TOKEN"
	default:
		return ""
	}
}
