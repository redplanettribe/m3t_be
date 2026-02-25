package postgres

import (
	"context"
	"database/sql"
	"strings"

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

// escapeILIKE escapes % and _ for use inside ILIKE pattern (so they match literally).
func escapeILIKE(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
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

func (r *eventInvitationRepository) ListByEventID(ctx context.Context, eventID string, search string, params domain.PaginationParams) ([]*domain.EventInvitation, int, error) {
	var total int
	if search != "" {
		pattern := "%" + escapeILIKE(search) + "%"
		countQuery := `
			SELECT COUNT(*)
			FROM event_invitations
			WHERE event_id = $1 AND email ILIKE $2
		`
		if err := r.DB.QueryRowContext(ctx, countQuery, eventID, pattern).Scan(&total); err != nil {
			return nil, 0, err
		}
	} else {
		countQuery := `
			SELECT COUNT(*)
			FROM event_invitations
			WHERE event_id = $1
		`
		if err := r.DB.QueryRowContext(ctx, countQuery, eventID).Scan(&total); err != nil {
			return nil, 0, err
		}
	}

	var query string
	var args []any
	if search != "" {
		pattern := "%" + escapeILIKE(search) + "%"
		query = `
			SELECT id, event_id, email, sent_at
			FROM event_invitations
			WHERE event_id = $1 AND email ILIKE $2
			ORDER BY sent_at DESC
			LIMIT $3 OFFSET $4
		`
		args = []any{eventID, pattern, params.PageSize, params.Offset()}
	} else {
		query = `
			SELECT id, event_id, email, sent_at
			FROM event_invitations
			WHERE event_id = $1
			ORDER BY sent_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []any{eventID, params.PageSize, params.Offset()}
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var invs []*domain.EventInvitation
	for rows.Next() {
		inv := &domain.EventInvitation{}
		if err := rows.Scan(&inv.ID, &inv.EventID, &inv.Email, &inv.SentAt); err != nil {
			return nil, 0, err
		}
		invs = append(invs, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if invs == nil {
		invs = []*domain.EventInvitation{}
	}
	return invs, total, nil
}
