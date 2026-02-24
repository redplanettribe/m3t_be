package postgres

import (
	"context"
	"database/sql"
	"errors"

	"multitrackticketing/internal/domain"
)

type eventRegistrationRepository struct {
	DB *sql.DB
}

func NewEventRegistrationRepository(db *sql.DB) domain.EventRegistrationRepository {
	return &eventRegistrationRepository{
		DB: db,
	}
}

func (r *eventRegistrationRepository) Create(ctx context.Context, reg *domain.EventRegistration) error {
	query := `
		INSERT INTO event_registrations (event_id, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	return r.DB.QueryRowContext(ctx, query, reg.EventID, reg.UserID, reg.CreatedAt, reg.UpdatedAt).
		Scan(&reg.ID)
}

func (r *eventRegistrationRepository) GetByEventAndUser(ctx context.Context, eventID, userID string) (*domain.EventRegistration, error) {
	query := `
		SELECT id, event_id, user_id, created_at, updated_at
		FROM event_registrations
		WHERE event_id = $1 AND user_id = $2
	`
	reg := &domain.EventRegistration{}
	err := r.DB.QueryRowContext(ctx, query, eventID, userID).
		Scan(&reg.ID, &reg.EventID, &reg.UserID, &reg.CreatedAt, &reg.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return reg, nil
}

func (r *eventRegistrationRepository) ListByUserID(ctx context.Context, userID string) ([]*domain.EventRegistration, error) {
	query := `
		SELECT id, event_id, user_id, created_at, updated_at
		FROM event_registrations
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regs []*domain.EventRegistration
	for rows.Next() {
		reg := &domain.EventRegistration{}
		if err := rows.Scan(&reg.ID, &reg.EventID, &reg.UserID, &reg.CreatedAt, &reg.UpdatedAt); err != nil {
			return nil, err
		}
		regs = append(regs, reg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if regs == nil {
		regs = []*domain.EventRegistration{}
	}
	return regs, nil
}

