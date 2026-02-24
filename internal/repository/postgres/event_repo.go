package postgres

import (
	"context"
	"database/sql"
	"errors"

	"multitrackticketing/internal/domain"
)

type eventRepository struct {
	DB *sql.DB
}

func NewEventRepository(db *sql.DB) domain.EventRepository {
	return &eventRepository{
		DB: db,
	}
}

func (r *eventRepository) Create(ctx context.Context, e *domain.Event) error {
	query := `
		INSERT INTO events (name, event_code, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	return r.DB.QueryRowContext(ctx, query, e.Name, e.EventCode, e.OwnerID, e.CreatedAt, e.UpdatedAt).Scan(&e.ID)
}

func (r *eventRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	query := `
		SELECT id, name, event_code, owner_id, created_at, updated_at
		FROM events
		WHERE id = $1
	`
	e := &domain.Event{}
	err := r.DB.QueryRowContext(ctx, query, id).Scan(&e.ID, &e.Name, &e.EventCode, &e.OwnerID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return e, nil
}

func (r *eventRepository) ListByOwnerID(ctx context.Context, ownerID string) ([]*domain.Event, error) {
	query := `
		SELECT id, name, event_code, owner_id, created_at, updated_at
		FROM events
		WHERE owner_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.DB.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]*domain.Event, 0)
	for rows.Next() {
		e := &domain.Event{}
		if err := rows.Scan(&e.ID, &e.Name, &e.EventCode, &e.OwnerID, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *eventRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM events WHERE id = $1`
	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}
