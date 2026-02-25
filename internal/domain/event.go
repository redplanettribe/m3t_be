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

// EventService defines the business logic for managing schedule
type EventService interface {
	CreateEvent(ctx context.Context, event *Event) error
	GetEventByID(ctx context.Context, eventID string) (*Event, []*Room, []*Session, error)
	ImportSessionizeData(ctx context.Context, eventID string, sessionizeID string) error
	ListEventsByOwner(ctx context.Context, ownerID string) ([]*Event, error)
	DeleteEvent(ctx context.Context, eventID string, ownerID string) error
	ToggleRoomNotBookable(ctx context.Context, eventID, roomID, ownerID string) (*Room, error)
	ListEventRooms(ctx context.Context, eventID, ownerID string) ([]*Room, error)
	GetEventRoom(ctx context.Context, eventID, roomID, ownerID string) (*Room, error)
	UpdateEventRoom(ctx context.Context, eventID, roomID, ownerID string, capacity int, description, howToGetThere string, notBookable *bool) (*Room, error)
	DeleteEventRoom(ctx context.Context, eventID, roomID, ownerID string) error
	AddEventTeamMember(ctx context.Context, eventID, userIDToAdd, ownerID string) error
	AddEventTeamMemberByEmail(ctx context.Context, eventID, email, ownerID string) (*EventTeamMember, error)
	ListEventTeamMembers(ctx context.Context, eventID, callerID string) ([]*EventTeamMember, error)
	RemoveEventTeamMember(ctx context.Context, eventID, userIDToRemove, ownerID string) error
	SendEventInvitations(ctx context.Context, eventID, ownerID string, emails []string) (sent int, failed []string, err error)
	ListEventInvitations(ctx context.Context, eventID, callerID string, search string, params PaginationParams) ([]*EventInvitation, int, error)
}

// EventRepository defines the interface for event storage
type EventRepository interface {
	Create(ctx context.Context, event *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	GetByEventCode(ctx context.Context, eventCode string) (*Event, error)
	ListByOwnerID(ctx context.Context, ownerID string) ([]*Event, error)
	Delete(ctx context.Context, id string) error
}
