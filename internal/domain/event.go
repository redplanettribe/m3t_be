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
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	EventCode   string     `json:"event_code"`
	OwnerID     string     `json:"owner_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Date        *time.Time `json:"date,omitempty"`
	Description *string    `json:"description,omitempty"`
	LocationLat *float64   `json:"location_lat,omitempty"`
	LocationLng *float64   `json:"location_lng,omitempty"`
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
	UpdateEvent(ctx context.Context, eventID, ownerID string, date *time.Time, description *string, locationLat, locationLng *float64) (*Event, error)
	CreateEventRoom(ctx context.Context, eventID, ownerID, name string, capacity int, description, howToGetThere string, notBookable bool) (*Room, error)
	CreateEventSession(ctx context.Context, eventID, ownerID, roomID, title, description string, startTime, endTime time.Time, tagNames, speakerIDs []string) (*Session, error)
	UpdateSessionSchedule(ctx context.Context, eventID, sessionID, ownerID string, roomID *string, startTime, endTime *time.Time) (*Session, error)
	UpdateSessionContent(ctx context.Context, eventID, sessionID, ownerID string, title *string, description *string) (*Session, error)
	ImportSessionizeData(ctx context.Context, eventID string, sessionizeID string) error
	ListEventsByOwner(ctx context.Context, ownerID string) ([]*Event, error)
	DeleteEvent(ctx context.Context, eventID string, ownerID string) error
	ToggleRoomNotBookable(ctx context.Context, eventID, roomID, ownerID string) (*Room, error)
	ListEventRooms(ctx context.Context, eventID, ownerID string) ([]*Room, error)
	GetEventRoom(ctx context.Context, eventID, roomID, ownerID string) (*Room, error)
	UpdateEventRoom(ctx context.Context, eventID, roomID, ownerID string, capacity int, description, howToGetThere string, notBookable *bool) (*Room, error)
	DeleteEventRoom(ctx context.Context, eventID, roomID, ownerID string) error
	DeleteEventSession(ctx context.Context, eventID, sessionID, ownerID string) error
	ListEventSpeakers(ctx context.Context, eventID, ownerID string) ([]*Speaker, error)
	GetEventSpeaker(ctx context.Context, eventID, speakerID, ownerID string) (*Speaker, []*Session, error)
	DeleteEventSpeaker(ctx context.Context, eventID, speakerID, ownerID string) error
	CreateEventSpeaker(ctx context.Context, eventID, ownerID string, firstName, lastName, bio, tagLine, profilePicture string, isTopSpeaker bool) (*Speaker, error)
	AddEventTeamMember(ctx context.Context, eventID, userIDToAdd, ownerID string) error
	AddEventTeamMemberByEmail(ctx context.Context, eventID, email, ownerID string) (*EventTeamMember, error)
	ListEventTeamMembers(ctx context.Context, eventID, callerID string) ([]*EventTeamMember, error)
	RemoveEventTeamMember(ctx context.Context, eventID, userIDToRemove, ownerID string) error
	SendEventInvitations(ctx context.Context, eventID, ownerID string, emails []string) (sent int, failed []string, err error)
	ListEventInvitations(ctx context.Context, eventID, callerID string, search string, params PaginationParams) ([]*EventInvitation, int, error)
	ListEventTags(ctx context.Context, eventID, callerID string) ([]*Tag, error)
	AddEventTags(ctx context.Context, eventID, ownerID string, tagNames []string) ([]*Tag, error)
	AddSessionTag(ctx context.Context, eventID, sessionID, ownerID, tagID string) error
	RemoveSessionTag(ctx context.Context, eventID, sessionID, ownerID, tagID string) error
	UpdateEventTag(ctx context.Context, eventID, tagID, ownerID, name string) (*Tag, error)
	RemoveEventTag(ctx context.Context, eventID, ownerID, tagID string) error
}

// EventRepository defines the interface for event storage
type EventRepository interface {
	Create(ctx context.Context, event *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	GetByEventCode(ctx context.Context, eventCode string) (*Event, error)
	ListByOwnerID(ctx context.Context, ownerID string) ([]*Event, error)
	Update(ctx context.Context, eventID string, date *time.Time, description *string, locationLat, locationLng *float64) (*Event, error)
	Delete(ctx context.Context, id string) error
}
