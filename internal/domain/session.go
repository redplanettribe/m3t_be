package domain

import (
	"context"
	"strings"
	"time"
)

// Room represents a physical room or track at the event
// swagger:model Room
type Room struct {
	ID              string    `json:"id"`
	EventID         string    `json:"event_id"`
	Name            string    `json:"name"`
	SourceSessionID int       `json:"source_session_id"`
	Source          string    `json:"source"`
	NotBookable     bool      `json:"not_bookable"`
	Capacity        int       `json:"capacity"`
	Description     string    `json:"description"`
	HowToGetThere   string    `json:"how_to_get_there"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// NewRoom returns a new Room with the given fields. ID is typically set by the repository on create.
// capacity, description, and howToGetThere default to 0/empty for Session imports.
func NewRoom(eventID, name string, sourceSessionID int, source string, notBookable bool, capacity int, description, howToGetThere string, createdAt, updatedAt time.Time) *Room {
	return &Room{
		EventID:         eventID,
		Name:            name,
		SourceSessionID: sourceSessionID,
		Source:          source,
		NotBookable:     notBookable,
		Capacity:        capacity,
		Description:     description,
		HowToGetThere:   howToGetThere,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
}

// Session represents a conference session or talk
// swagger:model Session
type Session struct {
	ID              string    `json:"id"`
	RoomID          string    `json:"room_id"`
	SourceSessionID string    `json:"source_session_id"`
	Source          string    `json:"source"`
	Title           string    `json:"title"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	Description     string    `json:"description"`
	// Tags are the tags associated with this session. Each tag includes both its ID and name.
	Tags       []*Tag   `json:"tags"`
	SpeakerIDs []string `json:"speaker_ids"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewSession returns a new Session with the given fields. ID is typically set by the repository on create.
// Tag names may be provided; the repository and tag repository are responsible for persisting tag links and
// hydrating full Tag objects (including IDs) when sessions are loaded from the database.
func NewSession(roomID, sourceSessionID, source, title, description string, startTime, endTime time.Time, tags []string, createdAt, updatedAt time.Time) *Session {
	var tagObjs []*Tag
	for _, name := range tags {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		tagObjs = append(tagObjs, &Tag{Name: name})
	}
	return &Session{
		RoomID:          roomID,
		SourceSessionID: sourceSessionID,
		Source:          source,
		Title:           title,
		StartTime:       startTime,
		EndTime:         endTime,
		Description:     description,
		Tags:            tagObjs,
		SpeakerIDs:      []string{},
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
}

// SessionRepository defines the interface for session, room, and speaker storage
type SessionRepository interface {
	CreateRoom(ctx context.Context, room *Room) error
	CreateSession(ctx context.Context, session *Session) error
	CreateSpeaker(ctx context.Context, speaker *Speaker) error
	CreateSessionSpeaker(ctx context.Context, sessionID, speakerID string) error
	DeleteScheduleByEventID(ctx context.Context, eventID string) error
	DeleteSpeakersByEventID(ctx context.Context, eventID string) error
	GetSessionByID(ctx context.Context, sessionID string) (*Session, error)
	GetRoomByID(ctx context.Context, roomID string) (*Room, error)
	ListRoomsByEventID(ctx context.Context, eventID string) ([]*Room, error)
	ListSessionsByEventID(ctx context.Context, eventID string) ([]*Session, error)
	ListSpeakerIDsBySessionIDs(ctx context.Context, sessionIDs []string) (map[string][]string, error)
	GetSpeakerByID(ctx context.Context, speakerID string) (*Speaker, error)
	ListSpeakersByEventID(ctx context.Context, eventID string) ([]*Speaker, error)
	ListSessionIDsBySpeakerID(ctx context.Context, speakerID string) ([]string, error)
	ListSessionsByIDs(ctx context.Context, sessionIDs []string) ([]*Session, error)
	DeleteSpeaker(ctx context.Context, speakerID string) error
	SetRoomNotBookable(ctx context.Context, roomID string, notBookable bool) (*Room, error)
	UpdateRoomDetails(ctx context.Context, roomID string, name string, capacity int, description, howToGetThere string, notBookable bool) (*Room, error)
	DeleteRoom(ctx context.Context, roomID string) error
	DeleteSession(ctx context.Context, sessionID string) error
	UpdateSessionSchedule(ctx context.Context, sessionID string, roomID *string, startTime, endTime *time.Time) (*Session, error)
	UpdateSessionContent(ctx context.Context, sessionID string, title *string, description *string) (*Session, error)
}
