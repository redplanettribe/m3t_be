package postgres

import (
	"context"
	"database/sql"
	"multitrackticketing/internal/domain"
	"time"

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
		INSERT INTO rooms (event_id, name, source_session_id, source, not_bookable, capacity, description, how_to_get_there, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (event_id, source_session_id) DO UPDATE 
		SET name = EXCLUDED.name, source = EXCLUDED.source, not_bookable = EXCLUDED.not_bookable, capacity = EXCLUDED.capacity, description = EXCLUDED.description, how_to_get_there = EXCLUDED.how_to_get_there, updated_at = EXCLUDED.updated_at
		RETURNING id
	`
	return r.DB.QueryRowContext(ctx, query, room.EventID, room.Name, room.SourceSessionID, room.Source, room.NotBookable, room.Capacity, room.Description, room.HowToGetThere, room.CreatedAt, room.UpdatedAt).Scan(&room.ID)
}

func (r *SessionRepository) CreateSession(ctx context.Context, s *domain.Session) error {
	query := `
		INSERT INTO sessions (room_id, source_session_id, source, title, start_time, end_time, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (room_id, source_session_id) DO UPDATE 
		SET source = EXCLUDED.source, title = EXCLUDED.title, start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time, description = EXCLUDED.description, updated_at = EXCLUDED.updated_at
		RETURNING id
	`
	return r.DB.QueryRowContext(ctx, query, s.RoomID, s.SourceSessionID, s.Source, s.Title, s.StartTime, s.EndTime, s.Description, s.CreatedAt, s.UpdatedAt).Scan(&s.ID)
}

func (r *SessionRepository) CreateSpeaker(ctx context.Context, speaker *domain.Speaker) error {
	query := `
		INSERT INTO speakers (event_id, source_session_id, source, first_name, last_name, full_name, bio, tag_line, profile_picture, is_top_speaker, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (event_id, source_session_id) DO UPDATE
		SET source = EXCLUDED.source, first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name, full_name = EXCLUDED.full_name, bio = EXCLUDED.bio, tag_line = EXCLUDED.tag_line, profile_picture = EXCLUDED.profile_picture, is_top_speaker = EXCLUDED.is_top_speaker, updated_at = EXCLUDED.updated_at
		RETURNING id
	`
	return r.DB.QueryRowContext(ctx, query,
		speaker.EventID, speaker.SourceSessionID, speaker.Source, speaker.FirstName, speaker.LastName, speaker.FullName,
		speaker.Bio, speaker.TagLine, speaker.ProfilePicture, speaker.IsTopSpeaker, speaker.CreatedAt, speaker.UpdatedAt,
	).Scan(&speaker.ID)
}

func (r *SessionRepository) CreateSessionSpeaker(ctx context.Context, sessionID, speakerID string) error {
	query := `INSERT INTO session_speakers (session_id, speaker_id) VALUES ($1, $2) ON CONFLICT (session_id, speaker_id) DO NOTHING`
	_, err := r.DB.ExecContext(ctx, query, sessionID, speakerID)
	return err
}

func (r *SessionRepository) DeleteScheduleByEventID(ctx context.Context, eventID string) error {
	query := `DELETE FROM rooms WHERE event_id = $1`
	_, err := r.DB.ExecContext(ctx, query, eventID)
	return err
}

func (r *SessionRepository) DeleteSpeakersByEventID(ctx context.Context, eventID string) error {
	query := `DELETE FROM speakers WHERE event_id = $1`
	_, err := r.DB.ExecContext(ctx, query, eventID)
	return err
}

func (r *SessionRepository) GetRoomByID(ctx context.Context, roomID string) (*domain.Room, error) {
	query := `
		SELECT id, event_id, name, source_session_id, source, not_bookable, capacity, description, how_to_get_there, created_at, updated_at
		FROM rooms
		WHERE id = $1
	`
	room := &domain.Room{}
	err := r.DB.QueryRowContext(ctx, query, roomID).Scan(&room.ID, &room.EventID, &room.Name, &room.SourceSessionID, &room.Source, &room.NotBookable, &room.Capacity, &room.Description, &room.HowToGetThere, &room.CreatedAt, &room.UpdatedAt)
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
		SELECT id, event_id, name, source_session_id, source, not_bookable, capacity, description, how_to_get_there, created_at, updated_at
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
		if err := rows.Scan(&room.ID, &room.EventID, &room.Name, &room.SourceSessionID, &room.Source, &room.NotBookable, &room.Capacity, &room.Description, &room.HowToGetThere, &room.CreatedAt, &room.UpdatedAt); err != nil {
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
		RETURNING id, event_id, name, source_session_id, source, not_bookable, capacity, description, how_to_get_there, created_at, updated_at
	`
	room := &domain.Room{}
	err := r.DB.QueryRowContext(ctx, query, roomID, notBookable).Scan(&room.ID, &room.EventID, &room.Name, &room.SourceSessionID, &room.Source, &room.NotBookable, &room.Capacity, &room.Description, &room.HowToGetThere, &room.CreatedAt, &room.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return room, nil
}

func (r *SessionRepository) UpdateRoomDetails(ctx context.Context, roomID string, capacity int, description, howToGetThere string, notBookable bool) (*domain.Room, error) {
	query := `
		UPDATE rooms
		SET capacity = $2, description = $3, how_to_get_there = $4, not_bookable = $5, updated_at = NOW()
		WHERE id = $1
		RETURNING id, event_id, name, source_session_id, source, not_bookable, capacity, description, how_to_get_there, created_at, updated_at
	`
	room := &domain.Room{}
	err := r.DB.QueryRowContext(ctx, query, roomID, capacity, description, howToGetThere, notBookable).Scan(&room.ID, &room.EventID, &room.Name, &room.SourceSessionID, &room.Source, &room.NotBookable, &room.Capacity, &room.Description, &room.HowToGetThere, &room.CreatedAt, &room.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return room, nil
}

func (r *SessionRepository) DeleteRoom(ctx context.Context, roomID string) error {
	result, err := r.DB.ExecContext(ctx, `DELETE FROM rooms WHERE id = $1`, roomID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SessionRepository) DeleteSession(ctx context.Context, sessionID string) error {
	result, err := r.DB.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SessionRepository) GetSessionByID(ctx context.Context, sessionID string) (*domain.Session, error) {
	query := `
		SELECT id, room_id, source_session_id, source, title, start_time, end_time, description, created_at, updated_at
		FROM sessions
		WHERE id = $1
	`
	sess := &domain.Session{}
	err := r.DB.QueryRowContext(ctx, query, sessionID).Scan(
		&sess.ID,
		&sess.RoomID,
		&sess.SourceSessionID,
		&sess.Source,
		&sess.Title,
		&sess.StartTime,
		&sess.EndTime,
		&sess.Description,
		&sess.CreatedAt,
		&sess.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	sess.Tags = []*domain.Tag{}
	rows, err := r.DB.QueryContext(ctx, `SELECT t.id, t.name FROM session_tags st JOIN tags t ON t.id = st.tag_id WHERE st.session_id = $1`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tagID, tagName string
		if err := rows.Scan(&tagID, &tagName); err != nil {
			return nil, err
		}
		sess.Tags = append(sess.Tags, &domain.Tag{ID: tagID, Name: tagName})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sess.SpeakerIDs = []string{}
	speakerMap, err := r.ListSpeakerIDsBySessionIDs(ctx, []string{sessionID})
	if err != nil {
		return nil, err
	}
	if ids := speakerMap[sessionID]; len(ids) > 0 {
		sess.SpeakerIDs = ids
	}
	return sess, nil
}

func (r *SessionRepository) ListSessionsByEventID(ctx context.Context, eventID string) ([]*domain.Session, error) {
	query := `
		SELECT s.id, s.room_id, s.source_session_id, s.source, s.title, s.start_time, s.end_time, s.description, s.created_at, s.updated_at
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
		if err := rows.Scan(&sess.ID, &sess.RoomID, &sess.SourceSessionID, &sess.Source, &sess.Title, &sess.StartTime, &sess.EndTime, &sess.Description, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
			return nil, err
		}
		sess.Tags = []*domain.Tag{}
		sess.SpeakerIDs = []string{}
		sessions = append(sessions, sess)
		sessionIDs = append(sessionIDs, sess.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(sessionIDs) == 0 {
		return sessions, nil
	}
	tagRows, err := r.DB.QueryContext(ctx, `SELECT st.session_id, t.id, t.name FROM session_tags st JOIN tags t ON t.id = st.tag_id WHERE st.session_id = ANY($1)`, pq.Array(sessionIDs))
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()
	tagsBySession := make(map[string][]*domain.Tag)
	for tagRows.Next() {
		var sid, tagID, tagName string
		if err := tagRows.Scan(&sid, &tagID, &tagName); err != nil {
			return nil, err
		}
		tagsBySession[sid] = append(tagsBySession[sid], &domain.Tag{ID: tagID, Name: tagName})
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

// ListSpeakerIDsBySessionIDs returns for each session ID the list of speaker IDs (order preserved).
func (r *SessionRepository) ListSpeakerIDsBySessionIDs(ctx context.Context, sessionIDs []string) (map[string][]string, error) {
	if len(sessionIDs) == 0 {
		return map[string][]string{}, nil
	}
	rows, err := r.DB.QueryContext(ctx, `SELECT session_id, speaker_id FROM session_speakers WHERE session_id = ANY($1) ORDER BY session_id, speaker_id`, pq.Array(sessionIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]string)
	for rows.Next() {
		var sessionID, speakerID string
		if err := rows.Scan(&sessionID, &speakerID); err != nil {
			return nil, err
		}
		out[sessionID] = append(out[sessionID], speakerID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SessionRepository) GetSpeakerByID(ctx context.Context, speakerID string) (*domain.Speaker, error) {
	query := `
		SELECT id, event_id, source_session_id, source, first_name, last_name, full_name, bio, tag_line, profile_picture, is_top_speaker, created_at, updated_at
		FROM speakers
		WHERE id = $1
	`
	sp := &domain.Speaker{}
	err := r.DB.QueryRowContext(ctx, query, speakerID).Scan(
		&sp.ID,
		&sp.EventID,
		&sp.SourceSessionID,
		&sp.Source,
		&sp.FirstName,
		&sp.LastName,
		&sp.FullName,
		&sp.Bio,
		&sp.TagLine,
		&sp.ProfilePicture,
		&sp.IsTopSpeaker,
		&sp.CreatedAt,
		&sp.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return sp, nil
}

func (r *SessionRepository) ListSpeakersByEventID(ctx context.Context, eventID string) ([]*domain.Speaker, error) {
	query := `
		SELECT id, event_id, source_session_id, source, first_name, last_name, full_name, bio, tag_line, profile_picture, is_top_speaker, created_at, updated_at
		FROM speakers
		WHERE event_id = $1
		ORDER BY full_name, id
	`
	rows, err := r.DB.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var speakers []*domain.Speaker
	for rows.Next() {
		sp := &domain.Speaker{}
		if err := rows.Scan(&sp.ID, &sp.EventID, &sp.SourceSessionID, &sp.Source, &sp.FirstName, &sp.LastName, &sp.FullName, &sp.Bio, &sp.TagLine, &sp.ProfilePicture, &sp.IsTopSpeaker, &sp.CreatedAt, &sp.UpdatedAt); err != nil {
			return nil, err
		}
		speakers = append(speakers, sp)
	}
	return speakers, rows.Err()
}

func (r *SessionRepository) ListSessionIDsBySpeakerID(ctx context.Context, speakerID string) ([]string, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT session_id FROM session_speakers WHERE speaker_id = $1 ORDER BY session_id`, speakerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *SessionRepository) ListSessionsByIDs(ctx context.Context, sessionIDs []string) ([]*domain.Session, error) {
	if len(sessionIDs) == 0 {
		return []*domain.Session{}, nil
	}
	query := `
		SELECT id, room_id, source_session_id, source, title, start_time, end_time, description, created_at, updated_at
		FROM sessions
		WHERE id = ANY($1)
		ORDER BY start_time, id
	`
	rows, err := r.DB.QueryContext(ctx, query, pq.Array(sessionIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []*domain.Session
	for rows.Next() {
		sess := &domain.Session{}
		if err := rows.Scan(&sess.ID, &sess.RoomID, &sess.SourceSessionID, &sess.Source, &sess.Title, &sess.StartTime, &sess.EndTime, &sess.Description, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
			return nil, err
		}
		sess.Tags = []*domain.Tag{}
		sess.SpeakerIDs = []string{}
		sessions = append(sessions, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	tagRows, err := r.DB.QueryContext(ctx, `SELECT st.session_id, t.id, t.name FROM session_tags st JOIN tags t ON t.id = st.tag_id WHERE st.session_id = ANY($1)`, pq.Array(sessionIDs))
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()
	tagsBySession := make(map[string][]*domain.Tag)
	for tagRows.Next() {
		var sid, tagID, tagName string
		if err := tagRows.Scan(&sid, &tagID, &tagName); err != nil {
			return nil, err
		}
		tagsBySession[sid] = append(tagsBySession[sid], &domain.Tag{ID: tagID, Name: tagName})
	}
	if err := tagRows.Err(); err != nil {
		return nil, err
	}
	speakerMap, err := r.ListSpeakerIDsBySessionIDs(ctx, sessionIDs)
	if err != nil {
		return nil, err
	}
	for _, sess := range sessions {
		if t := tagsBySession[sess.ID]; t != nil {
			sess.Tags = t
		}
		if ids := speakerMap[sess.ID]; len(ids) > 0 {
			sess.SpeakerIDs = ids
		}
	}
	return sessions, nil
}

func (r *SessionRepository) DeleteSpeaker(ctx context.Context, speakerID string) error {
	result, err := r.DB.ExecContext(ctx, `DELETE FROM speakers WHERE id = $1`, speakerID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SessionRepository) UpdateSessionSchedule(ctx context.Context, sessionID string, roomID *string, startTime, endTime *time.Time) (*domain.Session, error) {
	query := `
		UPDATE sessions
		SET
			room_id = COALESCE($2, room_id),
			start_time = COALESCE($3, start_time),
			end_time = COALESCE($4, end_time),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, room_id, source_session_id, source, title, start_time, end_time, description, created_at, updated_at
	`
	sess := &domain.Session{}
	err := r.DB.QueryRowContext(ctx, query, sessionID, roomID, startTime, endTime).Scan(
		&sess.ID,
		&sess.RoomID,
		&sess.SourceSessionID,
		&sess.Source,
		&sess.Title,
		&sess.StartTime,
		&sess.EndTime,
		&sess.Description,
		&sess.CreatedAt,
		&sess.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	sess.Tags = []*domain.Tag{}
	rows, err := r.DB.QueryContext(ctx, `SELECT t.id, t.name FROM session_tags st JOIN tags t ON t.id = st.tag_id WHERE st.session_id = $1`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tagID, tagName string
		if err := rows.Scan(&tagID, &tagName); err != nil {
			return nil, err
		}
		sess.Tags = append(sess.Tags, &domain.Tag{ID: tagID, Name: tagName})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sess, nil
}

func (r *SessionRepository) UpdateSessionContent(ctx context.Context, sessionID string, title *string, description *string) (*domain.Session, error) {
	query := `
		UPDATE sessions
		SET
			title = COALESCE($2, title),
			description = COALESCE($3, description),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, room_id, source_session_id, source, title, start_time, end_time, description, created_at, updated_at
	`
	sess := &domain.Session{}
	err := r.DB.QueryRowContext(ctx, query, sessionID, title, description).Scan(
		&sess.ID,
		&sess.RoomID,
		&sess.SourceSessionID,
		&sess.Source,
		&sess.Title,
		&sess.StartTime,
		&sess.EndTime,
		&sess.Description,
		&sess.CreatedAt,
		&sess.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	sess.Tags = []*domain.Tag{}
	rows, err := r.DB.QueryContext(ctx, `SELECT t.id, t.name FROM session_tags st JOIN tags t ON t.id = st.tag_id WHERE st.session_id = $1`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tagID, tagName string
		if err := rows.Scan(&tagID, &tagName); err != nil {
			return nil, err
		}
		sess.Tags = append(sess.Tags, &domain.Tag{ID: tagID, Name: tagName})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sess, nil
}
