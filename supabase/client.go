package supabase

import (
	"bytes"
	"context"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	anonKey string
	http    *http.Client
}

func New(url, anonKey string) *Client {
	return &Client{
		baseURL: url,
		anonKey: anonKey,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) newRequest(method, path string, body []byte) (*http.Request, error) {
	url := c.baseURL + path
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequestWithContext(context.Background(), method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(context.Background(), method, url, nil)
	}
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", c.anonKey)
	req.Header.Set("Authorization", "Bearer "+c.anonKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")
	return req, nil
}

// Do performs the request with the client's HTTP client and returns the response.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.http.Do(req)
}