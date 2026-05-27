package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Oridjinnn/hi/models"
)

func (c *Client) GetUser(username string) (*models.User, error) {
	path := "/users/" + username
	if username == "" {
		path = "/user"
	}
	req, err := c.newRequest(http.MethodGet, path)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var ghUser struct {
		Login       string `json:"login"`
		HTMLURL     string `json:"html_url"`
		AvatarURL   string `json:"avatar_url"`
		Bio         string `json:"bio"`
		PublicRepos int    `json:"public_repos"`
		Followers   int    `json:"followers"`
		CreatedAt   string `json:"created_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	createdAt, _ := time.Parse(time.RFC3339, ghUser.CreatedAt)

	return &models.User{
		GitHubUsername: ghUser.Login,
		GitHubURL:      ghUser.HTMLURL,
		AvatarURL:      ghUser.AvatarURL,
		Bio:            ghUser.Bio,
		PublicRepos:    ghUser.PublicRepos,
		Followers:      ghUser.Followers,
		CreatedAt:      createdAt,
	}, nil
}

func (c *Client) GetCurrentUser() (*models.User, error) {
	user, err := c.GetUser("")
	if err != nil {
		return nil, err
	}
	if user != nil {
		user.GitHubUsername = c.username
	}
	return user, nil
}
