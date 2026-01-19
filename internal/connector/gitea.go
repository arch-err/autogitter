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

// GiteaConnector implements the Connector interface for Gitea
type GiteaConnector struct {
	host   string
	token  string
	client *http.Client
}

// GiteaRepo represents a repository from the Gitea API
type GiteaRepo struct {
	FullName string `json:"full_name"`
	Archived bool   `json:"archived"`
	Empty    bool   `json:"empty"`
}

// GiteaUser represents a user from the Gitea API
type GiteaUser struct {
	Login string `json:"login"`
}

// GiteaOrg represents an organization check response
type GiteaOrg struct {
	ID int `json:"id"`
}

// NewGiteaConnector creates a new Gitea connector
func NewGiteaConnector(host, token string) *GiteaConnector {
	return &GiteaConnector{
		host:  host,
		token: token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the connector name
func (g *GiteaConnector) Name() string {
	return "gitea"
}

// apiURL returns the API base URL for Gitea
func (g *GiteaConnector) apiURL() string {
	return fmt.Sprintf("https://%s/api/v1", g.host)
}

// doRequest performs an authenticated HTTP request
func (g *GiteaConnector) doRequest(ctx context.Context, method, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if g.token != "" {
		req.Header.Set("Authorization", "token "+g.token)
	}

	return g.client.Do(req)
}

// TestConnection verifies the token works
func (g *GiteaConnector) TestConnection(ctx context.Context) error {
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
func (g *GiteaConnector) ListRepos(ctx context.Context, userOrOrg string) ([]string, error) {
	// First, check if this is an organization
	isOrg, err := g.isOrganization(ctx, userOrOrg)
	if err != nil {
		return nil, err
	}

	var repos []string
	page := 1

	for {
		var url string
		if isOrg {
			url = fmt.Sprintf("%s/orgs/%s/repos?page=%d&limit=50", g.apiURL(), userOrOrg, page)
		} else {
			url = fmt.Sprintf("%s/users/%s/repos?page=%d&limit=50", g.apiURL(), userOrOrg, page)
		}

		pageRepos, err := g.fetchRepoPage(ctx, url)
		if err != nil {
			return nil, err
		}

		if len(pageRepos) == 0 {
			break
		}

		repos = append(repos, pageRepos...)
		page++
	}

	return repos, nil
}

// isOrganization checks if the target is an organization
func (g *GiteaConnector) isOrganization(ctx context.Context, name string) (bool, error) {
	url := fmt.Sprintf("%s/orgs/%s", g.apiURL(), name)
	resp, err := g.doRequest(ctx, "GET", url)
	if err != nil {
		return false, fmt.Errorf("failed to check organization: %w", err)
	}
	defer resp.Body.Close()

	// If we get a 200, it's an organization
	if resp.StatusCode == 200 {
		return true, nil
	}

	// If we get a 404, check if it's a user
	if resp.StatusCode == 404 {
		userURL := fmt.Sprintf("%s/users/%s", g.apiURL(), name)
		userResp, err := g.doRequest(ctx, "GET", userURL)
		if err != nil {
			return false, fmt.Errorf("failed to check user: %w", err)
		}
		defer userResp.Body.Close()

		if userResp.StatusCode == 404 {
			return false, fmt.Errorf("user or organization not found: %s", name)
		}
		return false, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("failed to check organization: %s", string(body))
}

// fetchRepoPage fetches a single page of repositories
func (g *GiteaConnector) fetchRepoPage(ctx context.Context, url string) ([]string, error) {
	resp, err := g.doRequest(ctx, "GET", url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch repos: %s", string(body))
	}

	var giteaRepos []GiteaRepo
	if err := json.NewDecoder(resp.Body).Decode(&giteaRepos); err != nil {
		return nil, fmt.Errorf("failed to decode repos: %w", err)
	}

	var repos []string
	for _, repo := range giteaRepos {
		// Skip archived and empty repos
		if repo.Archived || repo.Empty {
			continue
		}
		repos = append(repos, repo.FullName)
	}

	return repos, nil
}

// TokenGenerationURL returns the URL where users can generate tokens
func (g *GiteaConnector) TokenGenerationURL() string {
	return fmt.Sprintf("https://%s/user/settings/applications", strings.TrimSuffix(g.host, "/"))
}
