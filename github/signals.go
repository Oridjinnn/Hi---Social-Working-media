package github

import (
	"fmt"
	"strings"
)

// CountOpenSignalsByAuthor returns how many open signals belong to username.
func (c *Client) CountOpenSignalsByAuthor(username string) (int, error) {
	if username == "" {
		return 0, nil
	}
	signals, err := c.ListSignals(nil, 1)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, s := range signals {
		if strings.EqualFold(s.Author.GitHubUsername, username) {
			n++
		}
	}
	return n, nil
}

// CheckSignalLimit returns an error when the user is at or over their active signal limit.
func (c *Client) CheckSignalLimit(username string, limit int, tier string) error {
	count, err := c.CountOpenSignalsByAuthor(username)
	if err != nil {
		return err
	}
	if count >= limit {
		return fmt.Errorf("signal limit reached (%d/%d on %s tier) — run: hi upgrade", count, limit, tier)
	}
	return nil
}
