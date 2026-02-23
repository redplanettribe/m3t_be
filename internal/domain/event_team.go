package domain

import (
	"context"
	"errors"
)

// ErrAlreadyMember is returned when adding a user who is already a team member of the event.
var ErrAlreadyMember = errors.New("already a team member")

// ErrInvalidInput is returned when the request is invalid (e.g. adding the event owner as a team member).
var ErrInvalidInput = errors.New("invalid input")

// EventTeamMember represents a user who is a team member of an event (excluding the owner).
// swagger:model EventTeamMember
type EventTeamMember struct {
	EventID  string `json:"event_id"`
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	LastName string `json:"last_name"`
	Email    string `json:"email"`
}

// EventTeamMemberRepository defines the interface for event team member storage.
type EventTeamMemberRepository interface {
	Add(ctx context.Context, eventID, userID string) error
	ListByEventID(ctx context.Context, eventID string) ([]*EventTeamMember, error)
	Remove(ctx context.Context, eventID, userID string) error
}
