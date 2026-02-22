package domain

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a requested entity is not found (e.g. event by ID).
var ErrNotFound = errors.New("not found")

// Event represents a conference event
// swagger:model Event
type Event struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewEvent returns a new Event with the given fields. ID is typically set by the repository on create.
func NewEvent(name, slug, ownerID string, createdAt, updatedAt time.Time) *Event {
	return &Event{
		Name:      name,
		Slug:      slug,
		OwnerID:   ownerID,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

// EventRepository defines the interface for event storage
type EventRepository interface {
	Create(ctx context.Context, event *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	GetBySlug(ctx context.Context, slug string) (*Event, error)
	ListByOwnerID(ctx context.Context, ownerID string) ([]*Event, error)
}
