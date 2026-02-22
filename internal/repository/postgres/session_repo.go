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

func (r *SessionRepository) ListRoomsByEventID(ctx context.Context, eventID string) ([]*domain.Room, error) {
	query := `
		SELECT id, event_id, name, sessionize_room_id, created_at, updated_at
		FROM rooms
		WHERE event_id = $1
		ORDER BY name
	`
	rows, err := r.DB.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rooms []*domain.Room
	for rows.Next() {
		room := &domain.Room{}
		if err := rows.Scan(&room.ID, &room.EventID, &room.Name, &room.SessionizeRoomID, &room.CreatedAt, &room.UpdatedAt); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}

func (r *SessionRepository) ListSessionsByEventID(ctx context.Context, eventID string) ([]*domain.Session, error) {
	query := `
		SELECT s.id, s.room_id, s.sessionize_session_id, s.title, s.start_time, s.end_time, s.description, s.created_at, s.updated_at
		FROM sessions s
		INNER JOIN rooms r ON r.id = s.room_id
		WHERE r.event_id = $1
		ORDER BY s.start_time, s.room_id
	`
	rows, err := r.DB.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []*domain.Session
	for rows.Next() {
		sess := &domain.Session{}
		if err := rows.Scan(&sess.ID, &sess.RoomID, &sess.SessionizeSessionID, &sess.Title, &sess.StartTime, &sess.EndTime, &sess.Description, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}
