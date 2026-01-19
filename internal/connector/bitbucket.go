package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BitbucketConnector implements the Connector interface for Bitbucket Cloud
type BitbucketConnector struct {
	host   string
	token  string
	client *http.Client
}

// BitbucketRepo represents a repository from the Bitbucket API
type BitbucketRepo struct {
	FullName string `json:"full_name"`
	IsPrivate bool  `json:"is_private"`
}

// BitbucketRepoResponse represents the paginated response
type BitbucketRepoResponse struct {
	Values []BitbucketRepo `json:"values"`
	Next   string          `json:"next"`
}

// BitbucketUser represents a user from the Bitbucket API
type BitbucketUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// NewBitbucketConnector creates a new Bitbucket connector
func NewBitbucketConnector(host, token string) *BitbucketConnector {
	if host == "" {
		host = "bitbucket.org"
	}

	return &BitbucketConnector{
		host:  host,
		token: token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the connector name
func (b *BitbucketConnector) Name() string {
	return "bitbucket"
}

// apiURL returns the API base URL for Bitbucket
func (b *BitbucketConnector) apiURL() string {
	if b.host == "bitbucket.org" {
		return "https://api.bitbucket.org/2.0"
	}
	// Bitbucket Server/Data Center
	return fmt.Sprintf("https://%s/rest/api/1.0", b.host)
}

// doRequest performs an authenticated HTTP request
func (b *BitbucketConnector) doRequest(ctx context.Context, method, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if b.token != "" {
		req.Header.Set("Authorization", "Bearer "+b.token)
	}

	return b.client.Do(req)
}

// TestConnection verifies the credentials work
func (b *BitbucketConnector) TestConnection(ctx context.Context) error {
	var url string
	if b.host == "bitbucket.org" {
		url = fmt.Sprintf("%s/user", b.apiURL())
	} else {
		// Bitbucket Server uses different endpoint
		url = fmt.Sprintf("%s/application-properties", b.apiURL())
	}
	resp, err := b.doRequest(ctx, "GET", url)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("authentication failed: invalid credentials")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListRepos returns all repos for the configured workspace/user
func (b *BitbucketConnector) ListRepos(ctx context.Context, workspace string) ([]string, error) {
	var repos []string
	url := fmt.Sprintf("%s/repositories/%s?pagelen=100", b.apiURL(), workspace)

	for url != "" {
		pageRepos, nextURL, err := b.fetchRepoPage(ctx, url)
		if err != nil {
			return nil, err
		}

		repos = append(repos, pageRepos...)
		url = nextURL
	}

	return repos, nil
}

// fetchRepoPage fetches a single page of repositories
func (b *BitbucketConnector) fetchRepoPage(ctx context.Context, url string) ([]string, string, error) {
	resp, err := b.doRequest(ctx, "GET", url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch repos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("failed to fetch repos: %s", string(body))
	}

	var response BitbucketRepoResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, "", fmt.Errorf("failed to decode repos: %w", err)
	}

	var repos []string
	for _, repo := range response.Values {
		repos = append(repos, repo.FullName)
	}

	return repos, response.Next, nil
}

// TokenGenerationURL returns the URL where users can generate app passwords
func (b *BitbucketConnector) TokenGenerationURL() string {
	return "https://bitbucket.org/account/settings/app-passwords/"
}
