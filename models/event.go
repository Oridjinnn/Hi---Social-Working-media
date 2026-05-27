package models

import "time"

type EventType string

const (
	EventTypeConnect EventType = "connect"
	EventTypeView    EventType = "view"
	EventTypeStar    EventType = "star"
)

type OutcomeType string

const (
	OutcomeJoinedProject   OutcomeType = "joined_project"
	OutcomeHadCall         OutcomeType = "had_call"
	OutcomeOngoing         OutcomeType = "ongoing"
	OutcomeNothingCameOfIt OutcomeType = "nothing"
)

type ConnectionEvent struct {
	ID            string      `json:"id"`
	SignalID      int64       `json:"signal_id"`
	SignalAuthor  string      `json:"signal_author"`
	ActorUsername string      `json:"actor_username"`
	EventType     EventType   `json:"event_type"`
	CreatedAt     time.Time   `json:"created_at"`

	Outcome   OutcomeType `json:"outcome,omitempty"`
	OutcomeAt *time.Time  `json:"outcome_at,omitempty"`
}