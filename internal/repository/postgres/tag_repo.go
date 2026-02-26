package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"multitrackticketing/internal/domain"

	"github.com/lib/pq"
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

func (r *tagRepository) AddSessionTag(ctx context.Context, sessionID, tagID string) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO session_tags (session_id, tag_id) VALUES ($1, $2) ON CONFLICT (session_id, tag_id) DO NOTHING`, sessionID, tagID)
	return err
}

func (r *tagRepository) RemoveSessionTag(ctx context.Context, sessionID, tagID string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM session_tags WHERE session_id = $1 AND tag_id = $2`, sessionID, tagID)
	return err
}

func (r *tagRepository) RemoveEventTag(ctx context.Context, eventID, tagID string) error {
	_, err := r.DB.ExecContext(ctx,
		`DELETE FROM session_tags WHERE tag_id = $1 AND session_id IN (SELECT s.id FROM sessions s JOIN rooms r ON s.room_id = r.id WHERE r.event_id = $2)`,
		tagID, eventID)
	if err != nil {
		return err
	}
	result, err := r.DB.ExecContext(ctx, `DELETE FROM event_tags WHERE event_id = $1 AND tag_id = $2`, eventID, tagID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *tagRepository) UpdateTagName(ctx context.Context, tagID, name string) error {
	result, err := r.DB.ExecContext(ctx, `UPDATE tags SET name = $2 WHERE id = $1`, tagID, name)
	if err != nil {
		var perr *pq.Error
		if errors.As(err, &perr) && perr.Code == "23505" {
			return fmt.Errorf("tag name already exists: %s", name)
		}
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *tagRepository) GetTagByID(ctx context.Context, tagID string) (*domain.Tag, error) {
	var tag domain.Tag
	err := r.DB.QueryRowContext(ctx, `SELECT id, name FROM tags WHERE id = $1`, tagID).Scan(&tag.ID, &tag.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &tag, nil
}
