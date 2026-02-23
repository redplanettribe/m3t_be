package postgres

import (
	"context"
	"database/sql"
	"multitrackticketing/internal/domain"

	"github.com/lib/pq"
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
		INSERT INTO rooms (event_id, name, sessionize_room_id, not_bookable, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (event_id, sessionize_room_id) DO UPDATE 
		SET name = EXCLUDED.name, updated_at = EXCLUDED.updated_at
		RETURNING id
	`
	return r.DB.QueryRowContext(ctx, query, room.EventID, room.Name, room.SessionizeRoomID, room.NotBookable, room.CreatedAt, room.UpdatedAt).Scan(&room.ID)
}

func (r *SessionRepository) CreateSession(ctx context.Context, s *domain.Session) error {
	query := `
		INSERT INTO sessions (room_id, sessionize_session_id, title, start_time, end_time, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (room_id, sessionize_session_id) DO UPDATE 
		SET title = EXCLUDED.title, start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time, description = EXCLUDED.description, updated_at = EXCLUDED.updated_at
		RETURNING id
	`
	if err := r.DB.QueryRowContext(ctx, query, s.RoomID, s.SessionizeSessionID, s.Title, s.StartTime, s.EndTime, s.Description, s.CreatedAt, s.UpdatedAt).Scan(&s.ID); err != nil {
		return err
	}
	// Replace tags for this session (handles both new insert and ON CONFLICT update)
	if _, err := r.DB.ExecContext(ctx, `DELETE FROM session_tags WHERE session_id = $1`, s.ID); err != nil {
		return err
	}
	for _, tag := range s.Tags {
		if _, err := r.DB.ExecContext(ctx, `INSERT INTO session_tags (session_id, tag) VALUES ($1, $2)`, s.ID, tag); err != nil {
			return err
		}
	}
	return nil
}

func (r *SessionRepository) DeleteScheduleByEventID(ctx context.Context, eventID string) error {
	query := `DELETE FROM rooms WHERE event_id = $1`
	_, err := r.DB.ExecContext(ctx, query, eventID)
	return err
}

func (r *SessionRepository) GetRoomByID(ctx context.Context, roomID string) (*domain.Room, error) {
	query := `
		SELECT id, event_id, name, sessionize_room_id, not_bookable, created_at, updated_at
		FROM rooms
		WHERE id = $1
	`
	room := &domain.Room{}
	err := r.DB.QueryRowContext(ctx, query, roomID).Scan(&room.ID, &room.EventID, &room.Name, &room.SessionizeRoomID, &room.NotBookable, &room.CreatedAt, &room.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return room, nil
}

func (r *SessionRepository) ListRoomsByEventID(ctx context.Context, eventID string) ([]*domain.Room, error) {
	query := `
		SELECT id, event_id, name, sessionize_room_id, not_bookable, created_at, updated_at
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
		if err := rows.Scan(&room.ID, &room.EventID, &room.Name, &room.SessionizeRoomID, &room.NotBookable, &room.CreatedAt, &room.UpdatedAt); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}

func (r *SessionRepository) SetRoomNotBookable(ctx context.Context, roomID string, notBookable bool) (*domain.Room, error) {
	query := `
		UPDATE rooms
		SET not_bookable = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, event_id, name, sessionize_room_id, not_bookable, created_at, updated_at
	`
	room := &domain.Room{}
	err := r.DB.QueryRowContext(ctx, query, roomID, notBookable).Scan(&room.ID, &room.EventID, &room.Name, &room.SessionizeRoomID, &room.NotBookable, &room.CreatedAt, &room.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return room, nil
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
	var sessionIDs []string
	for rows.Next() {
		sess := &domain.Session{}
		if err := rows.Scan(&sess.ID, &sess.RoomID, &sess.SessionizeSessionID, &sess.Title, &sess.StartTime, &sess.EndTime, &sess.Description, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
			return nil, err
		}
		sess.Tags = []string{}
		sessions = append(sessions, sess)
		sessionIDs = append(sessionIDs, sess.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(sessionIDs) == 0 {
		return sessions, nil
	}
	tagRows, err := r.DB.QueryContext(ctx, `SELECT session_id, tag FROM session_tags WHERE session_id = ANY($1)`, pq.Array(sessionIDs))
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()
	tagsBySession := make(map[string][]string)
	for tagRows.Next() {
		var sessionID, tag string
		if err := tagRows.Scan(&sessionID, &tag); err != nil {
			return nil, err
		}
		tagsBySession[sessionID] = append(tagsBySession[sessionID], tag)
	}
	if err := tagRows.Err(); err != nil {
		return nil, err
	}
	for _, sess := range sessions {
		if t := tagsBySession[sess.ID]; t != nil {
			sess.Tags = t
		}
	}
	return sessions, nil
}
