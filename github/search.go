package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Oridjinnn/hi/models"
)

func (c *Client) FetchGhostSignals(stacks []string) ([]models.Signal, error) {
	lang := "go"
	if len(stacks) > 0 {
		lang = stacks[0]
	}

	query := fmt.Sprintf("language:%s pushed:>%s stars:>10",
		lang,
		time.Now().AddDate(0, -3, 0).Format("2006-01-02"),
	)

	apiURL := fmt.Sprintf("%s/search/repositories?q=%s&sort=stars&order=desc&per_page=5",
		BaseURL,
		url.QueryEscape(query),
	)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Items []struct {
			ID       int64                  `json:"id"`
			FullName string                 `json:"full_name"`
			HTMLURL  string                 `json:"html_url"`
			Owner    struct{ Login string } `json:"owner"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	signals := make([]models.Signal, len(res.Items))
	for i, it := range res.Items {
		signals[i] = models.Signal{
			ID:      it.ID,
			Title:   it.FullName,
			IsGhost: true,
			Project: it.FullName,
			Author:  models.User{GitHubUsername: it.Owner.Login},
			Stack:   []string{lang},
		}
	}
	return signals, nil
}
