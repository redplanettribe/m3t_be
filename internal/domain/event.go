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

// EventRepository defines the interface for event storage
type EventRepository interface {
	Create(ctx context.Context, event *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	GetBySlug(ctx context.Context, slug string) (*Event, error)
}
