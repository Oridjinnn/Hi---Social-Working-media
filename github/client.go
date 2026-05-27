package github

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/Oridjinnn/hi/config"
)

const BaseURL = "https://api.github.com"

type Client struct {
	httpClient *http.Client
	token      string
	repoOwner  string
	repoName   string
	username   string
}

func New(cfg *config.Config) *Client {
	repoOwner := cfg.HIRepoOwner
	repoName := cfg.HIRepoName
	// Fallback to env vars set at build time / runtime
	if repoOwner == "" {
		repoOwner = os.Getenv("HI_SIGNAL_REPO_OWNER")
	}
	if repoName == "" {
		repoName = os.Getenv("HI_SIGNAL_REPO_NAME")
	}
	// Production fallback — canonical HI signals repo
	if repoOwner == "" {
		repoOwner = "Oridjinnn"
	}
	if repoName == "" {
		repoName = "hi"
	}
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token:     cfg.GitHubToken,
		repoOwner: repoOwner,
		repoName:  repoName,
		username:  cfg.GitHubUsername,
	}
}

func (c *Client) newRequest(method, path string) (*http.Request, error) {
	url := BaseURL + path
	req, err := http.NewRequestWithContext(context.Background(), method, url, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "hi-cli/1.0")
	return req, nil
}

// Do performs the request with the client's HTTP client and returns the response.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}
