package domain

import (
	"context"
	"time"
)

// Event represents a conference event
type Event struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewEvent returns a new Event with the given fields. ID is typically set by the repository on create.
func NewEvent(name, slug string, createdAt, updatedAt time.Time) *Event {
	return &Event{
		Name:      name,
		Slug:      slug,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

// EventRepository defines the interface for event storage
type EventRepository interface {
	Create(ctx context.Context, event *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	GetBySlug(ctx context.Context, slug string) (*Event, error)
}
