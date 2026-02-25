package domain

import (
	"context"
	"time"
)

// EventInvitation represents an email invited to register for an event.
// swagger:model EventInvitation
type EventInvitation struct {
	ID      string    `json:"id"`
	EventID string    `json:"event_id"`
	Email   string    `json:"email"`
	SentAt  time.Time `json:"sent_at"`
}

// EventInvitationRepository defines storage operations for event invitations.
type EventInvitationRepository interface {
	Create(ctx context.Context, inv *EventInvitation) error
	ListByEventID(ctx context.Context, eventID string) ([]*EventInvitation, error)
}
