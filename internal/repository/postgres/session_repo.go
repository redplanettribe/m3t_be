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

func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	query := `
		INSERT INTO sessions (title, start_time, end_time, track_id, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err := r.DB.QueryRowContext(ctx, query, s.Title, s.StartTime, s.EndTime, s.TrackID, s.Description).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return err
	}
	return nil
}
