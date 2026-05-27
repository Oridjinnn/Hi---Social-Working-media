package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Oridjinnn/hi/models"
)

type githubIssue struct {
	Number    int64      `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	HTMLURL   string     `json:"html_url"`
	State     string     `json:"state"`
	Labels    []ghLabel  `json:"labels"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	User      ghUser     `json:"user"`
}

type ghLabel struct {
	Name string `json:"name"`
}

type ghUser struct {
	Login     string `json:"login"`
	HTMLURL   string `json:"html_url"`
	AvatarURL string `json:"avatar_url"`
}

type createIssueRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Labels    []string `json:"labels"`
}

func (c *Client) CreateSignal(s *models.Signal) (*models.Signal, error) {
	labels := buildLabels(s)
	body := buildIssueBody(s)

	reqBody := createIssueRequest{
		Title:  s.Title,
		Body:   body,
		Labels: labels,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	req, err := c.newRequest(http.MethodPost,
		fmt.Sprintf("/repos/%s/%s/issues", c.repoOwner, c.repoName))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("GitHub API error creating issue: %s", resp.Status)
	}

	var created githubIssue
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return issueToSignal(&created), nil
}

func (c *Client) ListSignals(labels []string, page int) ([]models.Signal, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues?state=open&per_page=50&page=%d", c.repoOwner, c.repoName, page)
	if len(labels) > 0 {
		path += "&labels=" + strings.Join(labels, ",")
	}

	req, err := c.newRequest(http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var issues []githubIssue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	signals := make([]models.Signal, 0, len(issues))
	for _, issue := range issues {
		signals = append(signals, *issueToSignal(&issue))
	}
	return signals, nil
}

func (c *Client) GetSignal(issueNumber int64) (*models.Signal, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d", c.repoOwner, c.repoName, issueNumber)
	req, err := c.newRequest(http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var issue githubIssue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return issueToSignal(&issue), nil
}

func (c *Client) UpdateSignalStatus(issueNumber int64, status models.SignalStatus) error {
	labels := []string{fmt.Sprintf("status:%s", status)}
	body := map[string]interface{}{
		"labels": labels,
	}
	return c.updateIssue(issueNumber, body)
}

func (c *Client) CloseSignal(issueNumber int64, reason models.SignalStatus) error {
	body := map[string]interface{}{
		"state":  "closed",
		"labels": []string{fmt.Sprintf("status:%s", reason)},
	}
	return c.updateIssue(issueNumber, body)
}

func (c *Client) AddConnectionComment(issueNumber int64, username string) error {
	comment := fmt.Sprintf("@%s connected via HI", username)
	bodyBytes, err := json.Marshal(map[string]string{"body": comment})
	if err != nil {
		return err
	}

	req, err := c.newRequest(http.MethodPost,
		fmt.Sprintf("/repos/%s/%s/issues/%d/comments", c.repoOwner, c.repoName, issueNumber))
	if err != nil {
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("GitHub API error adding comment: %s", resp.Status)
	}
	return nil
}

func (c *Client) updateIssue(issueNumber int64, body map[string]interface{}) error {
	bodyBytes, _ := json.Marshal(body)
	req, err := c.newRequest(http.MethodPatch,
		fmt.Sprintf("/repos/%s/%s/issues/%d", c.repoOwner, c.repoName, issueNumber))
	if err != nil {
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API error: %s", resp.Status)
	}
	return nil
}

func buildLabels(s *models.Signal) []string {
	labels := []string{
		fmt.Sprintf("type:%s", s.Type),
		fmt.Sprintf("status:%s", s.Status),
		fmt.Sprintf("commitment:%s", s.Commitment),
		fmt.Sprintf("difficulty:%s", s.Difficulty),
	}
	for _, stack := range s.Stack {
		labels = append(labels, fmt.Sprintf("stack:%s", stack))
	}
	for _, need := range s.Needs {
		labels = append(labels, fmt.Sprintf("need:%s", need))
	}
	return labels
}

func buildIssueBody(s *models.Signal) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Project\n%s\n\n", s.Project))
	b.WriteString(fmt.Sprintf("## Contact Method\n%s\n\n", s.Contact))
	if s.ContactURL != "" {
		b.WriteString(fmt.Sprintf("## Contact URL\n%s\n\n", s.ContactURL))
	}
	if len(s.Stack) > 0 {
		b.WriteString(fmt.Sprintf("## Stack\n%s\n\n", strings.Join(s.Stack, ", ")))
	}
	if len(s.Needs) > 0 {
		b.WriteString(fmt.Sprintf("## Needs\n%s\n\n", strings.Join(s.Needs, ", ")))
	}
	if len(s.Skills) > 0 {
		b.WriteString(fmt.Sprintf("## Skills\n%s\n\n", strings.Join(s.Skills, ", ")))
	}
	if s.Body != "" {
		b.WriteString(fmt.Sprintf("## Details\n%s\n", s.Body))
	}
	return b.String()
}

func issueToSignal(issue *githubIssue) *models.Signal {
	s := &models.Signal{
		ID:        issue.Number,
		GitHubURL: issue.HTMLURL,
		Title:     issue.Title,
		Body:      issue.Body,
		CreatedAt: issue.CreatedAt,
		UpdatedAt: issue.UpdatedAt,
		Status:    models.SignalStatusOpen,
		Type:      models.SignalTypeContributor,
		Commitment: models.CommitmentCasual,
		Difficulty: models.DifficultyIntermediate,
		Author: models.User{
			GitHubUsername: issue.User.Login,
			GitHubURL:      issue.User.HTMLURL,
			AvatarURL:      issue.User.AvatarURL,
		},
	}

	// Parse labels
	for _, label := range issue.Labels {
		switch {
		case strings.HasPrefix(label.Name, "type:"):
			s.Type = models.SignalType(strings.TrimPrefix(label.Name, "type:"))
		case strings.HasPrefix(label.Name, "status:"):
			s.Status = models.SignalStatus(strings.TrimPrefix(label.Name, "status:"))
		case strings.HasPrefix(label.Name, "commitment:"):
			s.Commitment = models.CommitmentLevel(strings.TrimPrefix(label.Name, "commitment:"))
		case strings.HasPrefix(label.Name, "difficulty:"):
			s.Difficulty = models.DifficultyLevel(strings.TrimPrefix(label.Name, "difficulty:"))
		case strings.HasPrefix(label.Name, "stack:"):
			s.Stack = append(s.Stack, strings.TrimPrefix(label.Name, "stack:"))
		case strings.HasPrefix(label.Name, "need:"):
			s.Needs = append(s.Needs, strings.TrimPrefix(label.Name, "need:"))
		}
	}

	// Parse body for project name and contact info
	lines := strings.Split(issue.Body, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "## Project"):
			if i+1 < len(lines) {
				s.Project = strings.TrimSpace(lines[i+1])
			}
		case strings.HasPrefix(trimmed, "## Contact Method"):
			if i+1 < len(lines) {
				s.Contact = models.ContactMethod(strings.TrimSpace(lines[i+1]))
			}
		case strings.HasPrefix(trimmed, "## Contact URL"):
			if i+1 < len(lines) {
				s.ContactURL = strings.TrimSpace(lines[i+1])
			}
		}
	}

	return s
}

func parseIssueNumber(s string) (int64, error) {
	s = strings.TrimPrefix(s, "#")
	s = strings.TrimSpace(s)
	return strconv.ParseInt(s, 10, 64)
}
// SearchSignals searches signals by text query and optional label filters.
// Uses GitHub Issues search API with title/body matching.
func (c *Client) SearchSignals(textQuery string, labels []string) ([]models.Signal, error) {
	// Build search query: search within this repo's issues
	q := fmt.Sprintf("repo:%s/%s is:issue is:open", c.repoOwner, c.repoName)
	if textQuery != "" {
		q += " " + textQuery
	}
	for _, l := range labels {
		q += " label:" + l
	}

	path := "/search/issues?q=" + strings.ReplaceAll(q, " ", "+") + "&per_page=30"
	req, err := c.newRequest(http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var result struct {
		Items []githubIssue `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	signals := make([]models.Signal, 0, len(result.Items))
	for _, issue := range result.Items {
		signals = append(signals, *issueToSignal(&issue))
	}
	return signals, nil
}
