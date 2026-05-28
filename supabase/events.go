package supabase

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Oridjinnn/hi/models"
)

func (c *Client) LogConnectionEvent(e *models.ConnectionEvent) error {
	body, _ := json.Marshal(e)
	req, err := c.newRequest(http.MethodPost, "/rest/v1/connection_events", body)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("Supabase API error: %s", resp.Status)
	}
	return nil
}

func (c *Client) LogViewEvent(signalID int64, viewer string) error {
	body := map[string]interface{}{
		"signal_id": signalID,
		"viewer":    viewer,
	}
	bodyBytes, _ := json.Marshal(body)
	req, err := c.newRequest(http.MethodPost, "/rest/v1/signal_views", bodyBytes)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("Supabase API error: %s", resp.Status)
	}
	return nil
}

func (c *Client) GetPendingNotifications(username string) ([]models.ConnectionEvent, error) {
	url := fmt.Sprintf("/rest/v1/connection_events?signal_author=eq.%s&order=created_at.desc&limit=20", username)
	req, err := c.newRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Supabase API error: %s", resp.Status)
	}

	var events []models.ConnectionEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return events, nil
}

func (c *Client) UpdateOutcome(eventID string, outcome models.OutcomeType) error {
	body := map[string]interface{}{
		"outcome":    outcome,
		"outcome_at": "now()",
	}
	bodyBytes, _ := json.Marshal(body)
	url := fmt.Sprintf("/rest/v1/connection_events?id=eq.%s", eventID)
	req, err := c.newRequest(http.MethodPatch, url, bodyBytes)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("Supabase API error: %s", resp.Status)
	}
	return nil
}

func (c *Client) GetUserStats(username string) (connectionCount, successCount int, err error) {
	url := fmt.Sprintf("/rest/v1/users?github_username=eq.%s", username)
	req, err := c.newRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("executing request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("Supabase API error: %s", resp.Status)
	}

	var users []struct {
		ConnectionCount int `json:"connection_count"`
		SuccessCount    int `json:"success_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return 0, 0, fmt.Errorf("decoding response: %w", err)
	}

	if len(users) > 0 {
		return users[0].ConnectionCount, users[0].SuccessCount, nil
	}
	return 0, 0, nil
}

func (c *Client) UpsertUser(username string) error {
	body := map[string]interface{}{
		"github_username": username,
	}
	bodyBytes, _ := json.Marshal(body)
	req, err := c.newRequest(http.MethodPost, "/rest/v1/users", bodyBytes)
	if err != nil {
		return err
	}
	// Use upsert
	req.Header.Set("Prefer", "resolution=merge-duplicates")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 300 && !strings.Contains(resp.Status, "201") && !strings.Contains(resp.Status, "200") {
		return fmt.Errorf("Supabase API error: %s", resp.Status)
	}
	return nil
}
