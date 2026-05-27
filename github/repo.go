package github

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Oridjinnn/hi/models"
)

type repoData struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	HTMLURL     string `json:"html_url"`
	Stars       int    `json:"stargazers_count"`
	Forks       int    `json:"forks_count"`
	OpenIssues  int    `json:"open_issues_count"`
	Language    string `json:"language"`
}

func (c *Client) GetRepo() (*models.RepoData, error) {
	path := fmt.Sprintf("/repos/%s/%s", c.repoOwner, c.repoName)
	req, err := c.newRequest(http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var data repoData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &models.RepoData{
		ID:          data.ID,
		Name:        data.Name,
		FullName:    data.FullName,
		Description: data.Description,
		HTMLURL:     data.HTMLURL,
		Stars:       data.Stars,
		Forks:       data.Forks,
		OpenIssues:  data.OpenIssues,
		Language:    data.Language,
	}, nil
}

func (c *Client) StarRepo() error {
	path := fmt.Sprintf("/user/starred/%s/%s", c.repoOwner, c.repoName)
	req, err := http.NewRequest(http.MethodPut, BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "hi-cli/1.0")
	req.Header.Set("Content-Length", "0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API error: %s", resp.Status)
	}
	return nil
}