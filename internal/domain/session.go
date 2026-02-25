package domain

import (
	"context"
	"time"
)

// Room represents a physical room or track at the event
// swagger:model Room
type Room struct {
	ID               string    `json:"id"`
	EventID          string    `json:"event_id"`
	Name             string    `json:"name"`
	SessionizeRoomID int       `json:"source_session_id"`
	NotBookable      bool      `json:"not_bookable"`
	Capacity         int       `json:"capacity"`
	Description      string    `json:"description"`
	HowToGetThere    string    `json:"how_to_get_there"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// NewRoom returns a new Room with the given fields. ID is typically set by the repository on create.
// capacity, description, and howToGetThere default to 0/empty for Session imports.
func NewRoom(eventID, name string, sessionRoomID int, notBookable bool, capacity int, description, howToGetThere string, createdAt, updatedAt time.Time) *Room {
	return &Room{
		EventID:          eventID,
		Name:             name,
		SessionizeRoomID: sessionRoomID,
		NotBookable:      notBookable,
		Capacity:         capacity,
		Description:      description,
		HowToGetThere:    howToGetThere,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
}

// Session represents a conference session or talk
// swagger:model Session
type Session struct {
	ID              string    `json:"id"`
	RoomID          string    `json:"room_id"`
	SourceSessionID string    `json:"source_session_id"`
	Title           string    `json:"title"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	Description     string    `json:"description"`
	Tags            []string  `json:"tags"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// NewSession returns a new Session with the given fields. ID is typically set by the repository on create.
// tags may be nil or empty; the repository will store them in session_tags.
func NewSession(roomID, sessionizeSessionID, title, description string, startTime, endTime time.Time, tags []string, createdAt, updatedAt time.Time) *Session {
	if tags == nil {
		tags = []string{}
	}
	return &Session{
		RoomID:          roomID,
		SourceSessionID: sessionizeSessionID,
		Title:           title,
		StartTime:       startTime,
		EndTime:         endTime,
		Description:     description,
		Tags:            tags,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
}

// SessionRepository defines the interface for session and room storage
type SessionRepository interface {
	CreateRoom(ctx context.Context, room *Room) error
	CreateSession(ctx context.Context, session *Session) error
	DeleteScheduleByEventID(ctx context.Context, eventID string) error
	GetRoomByID(ctx context.Context, roomID string) (*Room, error)
	ListRoomsByEventID(ctx context.Context, eventID string) ([]*Room, error)
	ListSessionsByEventID(ctx context.Context, eventID string) ([]*Session, error)
	SetRoomNotBookable(ctx context.Context, roomID string, notBookable bool) (*Room, error)
	UpdateRoomDetails(ctx context.Context, roomID string, capacity int, description, howToGetThere string, notBookable bool) (*Room, error)
	DeleteRoom(ctx context.Context, roomID string) error
}
