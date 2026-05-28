package supabase

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Oridjinnn/hi/models"
)

// RealtimeClient wraps Supabase “realtime” notifications.
// MVP: this implementation is polling (REST queries) instead of a true WebSocket subscription.
// A production version can be upgraded to Supabase Realtime WebSocket.

type RealtimeClient struct {
	client     *Client
	username   string
	notifyFn   func(models.ConnectionEvent)
	stopCh     chan struct{}
	pollTicker *time.Ticker
}

func (c *Client) Subscribe(username string, notifyFn func(models.ConnectionEvent)) (*RealtimeClient, error) {
	rt := &RealtimeClient{
		client:   c,
		username: username,
		notifyFn: notifyFn,
		stopCh:   make(chan struct{}),
	}

	go rt.poll()
	return rt, nil
}

func (rt *RealtimeClient) poll() {
	rt.pollTicker = time.NewTicker(30 * time.Second)
	defer rt.pollTicker.Stop()

	lastCheck := time.Now()

	for {
		select {
		case <-rt.stopCh:
			return
		case <-rt.pollTicker.C:
			events, err := rt.client.getEventsSince(rt.username, lastCheck)
			if err != nil {
				continue
			}
			for _, event := range events {
				if rt.notifyFn != nil {
					rt.notifyFn(event)
				}
			}
			lastCheck = time.Now()
		}
	}
}

func (rt *RealtimeClient) Unsubscribe() error {
	close(rt.stopCh)
	return nil
}

func (c *Client) getEventsSince(username string, since time.Time) ([]models.ConnectionEvent, error) {
	url := fmt.Sprintf("/rest/v1/connection_events?signal_author=eq.%s&created_at=gt.%s&order=created_at.asc",
		username, since.UTC().Format(time.RFC3339))

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
