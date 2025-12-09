package postgres

import (
	"context"
	"database/sql"
	"multitrackticketing/internal/domain"
)

type SessionRepository struct {
	DB *sql.DB
}

func NewSessionRepository(db *sql.DB) domain.SessionRepository {
	return &SessionRepository{
		DB: db,
	}
}

func (r *SessionRepository) CreateRoom(ctx context.Context, room *domain.Room) error {
	query := `
		INSERT INTO rooms (event_id, name, sessionize_room_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_id, sessionize_room_id) DO UPDATE 
		SET name = EXCLUDED.name, updated_at = EXCLUDED.updated_at
		RETURNING id
	`
	return r.DB.QueryRowContext(ctx, query, room.EventID, room.Name, room.SessionizeRoomID, room.CreatedAt, room.UpdatedAt).Scan(&room.ID)
}

func (r *SessionRepository) CreateSession(ctx context.Context, s *domain.Session) error {
	query := `
		INSERT INTO sessions (room_id, sessionize_session_id, title, start_time, end_time, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (room_id, sessionize_session_id) DO UPDATE 
		SET title = EXCLUDED.title, start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time, description = EXCLUDED.description, updated_at = EXCLUDED.updated_at
		RETURNING id
	`
	return r.DB.QueryRowContext(ctx, query, s.RoomID, s.SessionizeSessionID, s.Title, s.StartTime, s.EndTime, s.Description, s.CreatedAt, s.UpdatedAt).Scan(&s.ID)
}

func (r *SessionRepository) DeleteScheduleByEventID(ctx context.Context, eventID string) error {
	query := `DELETE FROM rooms WHERE event_id = $1`
	_, err := r.DB.ExecContext(ctx, query, eventID)
	return err
}
