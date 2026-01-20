package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GitHubConnector implements the Connector interface for GitHub
type GitHubConnector struct {
	host   string
	token  string
	client *http.Client
}

// GitHubRepo represents a repository from the GitHub API
type GitHubRepo struct {
	FullName string `json:"full_name"`
	Archived bool   `json:"archived"`
	Disabled bool   `json:"disabled"`
}

// GitHubUser represents a user from the GitHub API
type GitHubUser struct {
	Login string `json:"login"`
	Type  string `json:"type"` // "User" or "Organization"
}

// NewGitHubConnector creates a new GitHub connector
func NewGitHubConnector(host, token string) *GitHubConnector {
	if host == "" {
		host = "github.com"
	}
	return &GitHubConnector{
		host:  host,
		token: token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the connector name
func (g *GitHubConnector) Name() string {
	return "github"
}

// apiURL returns the API base URL for GitHub
func (g *GitHubConnector) apiURL() string {
	if g.host == "github.com" {
		return "https://api.github.com"
	}
	// GitHub Enterprise
	return fmt.Sprintf("https://%s/api/v3", g.host)
}

// doRequest performs an authenticated HTTP request
func (g *GitHubConnector) doRequest(ctx context.Context, method, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}

	return g.client.Do(req)
}

// TestConnection verifies the token works
func (g *GitHubConnector) TestConnection(ctx context.Context) error {
	url := fmt.Sprintf("%s/user", g.apiURL())
	resp, err := g.doRequest(ctx, "GET", url)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed: invalid token")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListRepos returns all repos for the configured user/org
func (g *GitHubConnector) ListRepos(ctx context.Context, userOrOrg string) ([]string, error) {
	// First, determine if this is a user or organization
	userType, err := g.getUserType(ctx, userOrOrg)
	if err != nil {
		return nil, err
	}

	var repos []string
	var url string
	page := 1

	for {
		if userType == "Organization" {
			url = fmt.Sprintf("%s/orgs/%s/repos?per_page=100&page=%d", g.apiURL(), userOrOrg, page)
		} else {
			url = fmt.Sprintf("%s/users/%s/repos?per_page=100&page=%d", g.apiURL(), userOrOrg, page)
		}

		pageRepos, hasMore, err := g.fetchRepoPage(ctx, url)
		if err != nil {
			return nil, err
		}

		repos = append(repos, pageRepos...)

		if !hasMore {
			break
		}
		page++
	}

	return repos, nil
}

// getUserType determines if the target is a user or organization
func (g *GitHubConnector) getUserType(ctx context.Context, userOrOrg string) (string, error) {
	url := fmt.Sprintf("%s/users/%s", g.apiURL(), userOrOrg)
	resp, err := g.doRequest(ctx, "GET", url)
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", fmt.Errorf("user or organization not found: %s", userOrOrg)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get user info: %s", string(body))
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("failed to decode user info: %w", err)
	}

	return user.Type, nil
}

// fetchRepoPage fetches a single page of repositories
func (g *GitHubConnector) fetchRepoPage(ctx context.Context, url string) ([]string, bool, error) {
	resp, err := g.doRequest(ctx, "GET", url)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch repos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("failed to fetch repos: %s", string(body))
	}

	var ghRepos []GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&ghRepos); err != nil {
		return nil, false, fmt.Errorf("failed to decode repos: %w", err)
	}

	var repos []string
	for _, repo := range ghRepos {
		// Skip archived and disabled repos
		if repo.Archived || repo.Disabled {
			continue
		}
		repos = append(repos, repo.FullName)
	}

	// Check for next page via Link header
	hasMore := false
	linkHeader := resp.Header.Get("Link")
	if linkHeader != "" {
		hasMore = strings.Contains(linkHeader, `rel="next"`)
	}

	return repos, hasMore, nil
}

