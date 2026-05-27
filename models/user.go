package models

import "time"

type User struct {
	GitHubUsername string    `json:"github_username"`
	GitHubURL      string    `json:"github_url"`
	AvatarURL      string    `json:"avatar_url"`
	Bio            string    `json:"bio"`
	PublicRepos    int       `json:"public_repos"`
	Followers      int       `json:"followers"`
	CreatedAt      time.Time `json:"created_at"`

	SignalCount     int  `json:"signal_count"`
	ConnectionCount int  `json:"connection_count"`
	SuccessCount    int  `json:"success_count"`
	IsSupporter     bool `json:"is_supporter"`
	IsSeedUser      bool `json:"is_seed_user"`
}
