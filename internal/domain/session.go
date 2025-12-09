package domain

import (
	"context"
	"time"
)

// Room represents a physical room or track at the event
type Room struct {
	ID               string    `json:"id"`
	EventID          string    `json:"event_id"`
	Name             string    `json:"name"`
	SessionizeRoomID int       `json:"sessionize_room_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Session represents a conference session or talk
type Session struct {
	ID                  string    `json:"id"`
	RoomID              string    `json:"room_id"`
	SessionizeSessionID string    `json:"sessionize_session_id"`
	Title               string    `json:"title"`
	StartTime           time.Time `json:"start_time"`
	EndTime             time.Time `json:"end_time"`
	Description         string    `json:"description"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// SessionRepository defines the interface for session and room storage
type SessionRepository interface {
	CreateRoom(ctx context.Context, room *Room) error
	CreateSession(ctx context.Context, session *Session) error
	DeleteScheduleByEventID(ctx context.Context, eventID string) error
	// GetSessionByID(ctx context.Context, id string) (*Session, error) // Future use
	// ListSessions(ctx context.Context) ([]*Session, error) // Future use
}

// ManageScheduleUseCase defines the business logic for managing schedule
type ManageScheduleUseCase interface {
	CreateEvent(ctx context.Context, event *Event) error
	ImportSessionizeData(ctx context.Context, eventID string, sessionizeID string) error
}
