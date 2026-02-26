package postgres

import (
	"context"
	"database/sql"
	"multitrackticketing/internal/domain"
)

type tagRepository struct {
	DB *sql.DB
}

// NewTagRepository returns a domain.TagRepository implemented with Postgres.
func NewTagRepository(db *sql.DB) domain.TagRepository {
	return &tagRepository{DB: db}
}

func (r *tagRepository) EnsureTagForEvent(ctx context.Context, eventID, tagName string) (string, error) {
	var tagID string
	err := r.DB.QueryRowContext(ctx, `SELECT id FROM tags WHERE name = $1`, tagName).Scan(&tagID)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	if err == sql.ErrNoRows {
		if err := r.DB.QueryRowContext(ctx, `INSERT INTO tags (name) VALUES ($1) RETURNING id`, tagName).Scan(&tagID); err != nil {
			return "", err
		}
	}
	_, err = r.DB.ExecContext(ctx, `INSERT INTO event_tags (event_id, tag_id) VALUES ($1, $2) ON CONFLICT (event_id, tag_id) DO NOTHING`, eventID, tagID)
	if err != nil {
		return "", err
	}
	return tagID, nil
}

func (r *tagRepository) ListTagsByEventID(ctx context.Context, eventID string) ([]*domain.Tag, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT t.id, t.name FROM tags t
		 JOIN event_tags et ON et.tag_id = t.id
		 WHERE et.event_id = $1
		 ORDER BY t.name`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*domain.Tag
	for rows.Next() {
		var tag domain.Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

func (r *tagRepository) SetSessionTags(ctx context.Context, sessionID string, tagIDs []string) error {
	if _, err := r.DB.ExecContext(ctx, `DELETE FROM session_tags WHERE session_id = $1`, sessionID); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if _, err := r.DB.ExecContext(ctx, `INSERT INTO session_tags (session_id, tag_id) VALUES ($1, $2) ON CONFLICT (session_id, tag_id) DO NOTHING`, sessionID, tagID); err != nil {
			return err
		}
	}
	return nil
}
