package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ChatMessage struct {
	User      string    `json:"user"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func (c *Client) PostComment(signalID int64, body string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", BaseURL, c.repoOwner, c.repoName, signalID)
	
	payload := map[string]string{"body": body}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("github api error: %s", resp.Status)
	}
	return nil
}

func (c *Client) PollComments(signalID int64, since time.Time) ([]ChatMessage, error) {
	sinceStr := since.UTC().Format(time.RFC3339)
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments?since=%s",
		BaseURL, c.repoOwner, c.repoName, signalID, sinceStr)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ghComments []struct {
		User      struct{ Login string } `json:"user"`
		Body      string                `json:"body"`
		CreatedAt time.Time             `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghComments); err != nil {
		return nil, err
	}

	messages := make([]ChatMessage, len(ghComments))
	for i, gc := range ghComments {
		messages[i] = ChatMessage{User: gc.User.Login, Body: gc.Body, CreatedAt: gc.CreatedAt}
	}
	return messages, nil
}

func (c *Client) ListComments(signalID int64) ([]ChatMessage, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", BaseURL, c.repoOwner, c.repoName, signalID)
	
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ghComments []struct {
		User      struct{ Login string } `json:"user"`
		Body      string                `json:"body"`
		CreatedAt time.Time             `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghComments); err != nil {
		return nil, err
	}

	messages := make([]ChatMessage, len(ghComments))
	for i, gc := range ghComments {
		messages[i] = ChatMessage{User: gc.User.Login, Body: gc.Body, CreatedAt: gc.CreatedAt}
	}
	return messages, nil
}