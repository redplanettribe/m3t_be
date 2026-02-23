package postgres

import (
	"context"
	"database/sql"

	"github.com/lib/pq"

	"multitrackticketing/internal/domain"
)

type eventTeamMemberRepository struct {
	DB *sql.DB
}

func NewEventTeamMemberRepository(db *sql.DB) domain.EventTeamMemberRepository {
	return &eventTeamMemberRepository{
		DB: db,
	}
}

func (r *eventTeamMemberRepository) Add(ctx context.Context, eventID, userID string) error {
	query := `
		INSERT INTO event_team_members (event_id, user_id)
		VALUES ($1, $2)
	`
	_, err := r.DB.ExecContext(ctx, query, eventID, userID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return domain.ErrAlreadyMember
		}
		return err
	}
	return nil
}

func (r *eventTeamMemberRepository) ListByEventID(ctx context.Context, eventID string) ([]*domain.EventTeamMember, error) {
	query := `
		SELECT e.event_id, e.user_id, u.name, u.last_name, u.email
		FROM event_team_members e
		JOIN users u ON u.id = e.user_id
		WHERE e.event_id = $1
		ORDER BY e.user_id
	`
	rows, err := r.DB.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := make([]*domain.EventTeamMember, 0)
	for rows.Next() {
		m := &domain.EventTeamMember{}
		var name, lastName sql.NullString
		if err := rows.Scan(&m.EventID, &m.UserID, &name, &lastName, &m.Email); err != nil {
			return nil, err
		}
		m.Name = name.String
		m.LastName = lastName.String
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *eventTeamMemberRepository) Remove(ctx context.Context, eventID, userID string) error {
	query := `DELETE FROM event_team_members WHERE event_id = $1 AND user_id = $2`
	result, err := r.DB.ExecContext(ctx, query, eventID, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}
