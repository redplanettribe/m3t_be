package domain

import (
	"context"
	"time"
)

// EventRegistration represents an attendee's registration for an event.
// swagger:model EventRegistration
type EventRegistration struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewEventRegistration creates a new EventRegistration. ID is typically set by the repository on create.
func NewEventRegistration(eventID, userID string, createdAt, updatedAt time.Time) *EventRegistration {
	return &EventRegistration{
		EventID:   eventID,
		UserID:    userID,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

// EventRegistrationRepository defines storage operations for event registrations.
type EventRegistrationRepository interface {
	Create(ctx context.Context, reg *EventRegistration) error
	GetByEventAndUser(ctx context.Context, eventID, userID string) (*EventRegistration, error)
	ListByUserID(ctx context.Context, userID string) ([]*EventRegistration, error)
}

// EventRegistrationWithEvent bundles a registration with its related event.
type EventRegistrationWithEvent struct {
	Registration *EventRegistration `json:"registration"`
	Event        *Event `json:"event"`
}

// AttendeeService defines attendee-facing operations such as event registration.
type AttendeeService interface {
	// RegisterForEvent registers the user for the event. Returns (reg, created, err): created is true if a new registration was created, false if already registered.
	RegisterForEvent(ctx context.Context, eventID, userID string) (*EventRegistration, bool, error)
	// RegisterForEventByCode registers the user for the event identified by event_code. Returns (reg, created, err): created is true if a new registration was created, false if already registered.
	RegisterForEventByCode(ctx context.Context, eventCode, userID string) (*EventRegistration, bool, error)
	ListMyRegisteredEvents(ctx context.Context, userID string) ([]*EventRegistrationWithEvent, error)
}

