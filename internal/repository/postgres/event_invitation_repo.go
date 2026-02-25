package postgres

import (
	"context"
	"database/sql"

	"multitrackticketing/internal/domain"
)

type eventInvitationRepository struct {
	DB *sql.DB
}

func NewEventInvitationRepository(db *sql.DB) domain.EventInvitationRepository {
	return &eventInvitationRepository{
		DB: db,
	}
}

func (r *eventInvitationRepository) Create(ctx context.Context, inv *domain.EventInvitation) error {
	query := `
		INSERT INTO event_invitations (event_id, email, sent_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	return r.DB.QueryRowContext(ctx, query, inv.EventID, inv.Email, inv.SentAt).
		Scan(&inv.ID)
}

func (r *eventInvitationRepository) ListByEventID(ctx context.Context, eventID string) ([]*domain.EventInvitation, error) {
	query := `
		SELECT id, event_id, email, sent_at
		FROM event_invitations
		WHERE event_id = $1
		ORDER BY sent_at DESC
	`
	rows, err := r.DB.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invs []*domain.EventInvitation
	for rows.Next() {
		inv := &domain.EventInvitation{}
		if err := rows.Scan(&inv.ID, &inv.EventID, &inv.Email, &inv.SentAt); err != nil {
			return nil, err
		}
		invs = append(invs, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if invs == nil {
		invs = []*domain.EventInvitation{}
	}
	return invs, nil
}
