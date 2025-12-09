package postgres

import (
	"context"
	"database/sql"
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
		INSERT INTO events (name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	return r.DB.QueryRowContext(ctx, query, e.Name, e.Slug, e.CreatedAt, e.UpdatedAt).Scan(&e.ID)
}

func (r *eventRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	query := `
		SELECT id, name, slug, created_at, updated_at
		FROM events
		WHERE id = $1
	`
	e := &domain.Event{}
	err := r.DB.QueryRowContext(ctx, query, id).Scan(&e.ID, &e.Name, &e.Slug, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *eventRepository) GetBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	query := `
		SELECT id, name, slug, created_at, updated_at
		FROM events
		WHERE slug = $1
	`
	e := &domain.Event{}
	err := r.DB.QueryRowContext(ctx, query, slug).Scan(&e.ID, &e.Name, &e.Slug, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return e, nil
}
