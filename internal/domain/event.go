package domain

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a requested entity is not found (e.g. event by ID).
var ErrNotFound = errors.New("not found")

// ErrForbidden is returned when the user is not allowed to perform the action (e.g. not the event owner).
var ErrForbidden = errors.New("forbidden")

// Event represents a conference event
// swagger:model Event
type Event struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	EventCode string    `json:"event_code"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewEvent returns a new Event with the given fields. ID is typically set by the repository on create.
func NewEvent(name, eventCode, ownerID string, createdAt, updatedAt time.Time) *Event {
	return &Event{
		Name:      name,
		EventCode: eventCode,
		OwnerID:   ownerID,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

// EventRepository defines the interface for event storage
type EventRepository interface {
	Create(ctx context.Context, event *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	ListByOwnerID(ctx context.Context, ownerID string) ([]*Event, error)
	Delete(ctx context.Context, id string) error
}
