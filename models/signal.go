package models

import "time"

type SignalType string

const (
	SignalTypeContributor SignalType = "contributor"
	SignalTypeBeginner    SignalType = "beginner"
	SignalTypeVibeCoder   SignalType = "vibe_coder"
	SignalTypeHiring      SignalType = "hiring"
	SignalTypeShowcase    SignalType = "showcase"
)

type CommitmentLevel string

const (
	CommitmentCasual   CommitmentLevel = "casual"
	CommitmentPartTime CommitmentLevel = "part-time"
	CommitmentFullTime CommitmentLevel = "full-time"
)

type DifficultyLevel string

const (
	DifficultyBeginner     DifficultyLevel = "beginner"
	DifficultyIntermediate DifficultyLevel = "intermediate"
	DifficultyAdvanced     DifficultyLevel = "advanced"
)

type SignalStatus string

const (
	SignalStatusOpen       SignalStatus = "open"
	SignalStatusInProgress SignalStatus = "in-progress"
	SignalStatusFilled     SignalStatus = "filled"
	SignalStatusPaused     SignalStatus = "paused"
	SignalStatusExpired    SignalStatus = "expired"
)

type ContactMethod string

const (
	ContactGitHub  ContactMethod = "github"
	ContactDiscord ContactMethod = "discord"
	ContactEmail   ContactMethod = "email"
	ContactMatrix  ContactMethod = "matrix"
)

type Signal struct {
	ID        int64     `json:"id"`
	GitHubURL string    `json:"github_url"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ExpiresAt time.Time `json:"expires_at"`

	Type       SignalType      `json:"type"`
	Status     SignalStatus    `json:"status"`
	Project    string          `json:"project"`
	Stack      []string        `json:"stack"`
	Needs      []string        `json:"needs"`
	Skills     []string        `json:"skills"`
	Difficulty DifficultyLevel `json:"difficulty"`
	Commitment CommitmentLevel `json:"commitment"`
	Contact    ContactMethod   `json:"contact_method"`
	ContactURL string          `json:"contact_url"`

	Author User `json:"author"`

	ViewCount    int `json:"view_count"`
	ConnectCount int `json:"connect_count"`

	IsGhost bool `json:"is_ghost,omitempty"`
}