package domain

import (
	"context"
	"time"
)

// Session represents a conference session or talk
type Session struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	TrackID     string    `json:"track_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SessionRepository defines the interface for session storage
type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	// GetByID(ctx context.Context, id string) (*Session, error) // Future use
	// List(ctx context.Context) ([]*Session, error) // Future use
}

// ManageScheduleUseCase defines the business logic for managing schedule
type ManageScheduleUseCase interface {
	CreateSession(ctx context.Context, session *Session) error
}
