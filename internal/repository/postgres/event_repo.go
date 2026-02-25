package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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
		SELECT id, name, event_code, owner_id, created_at, updated_at, date, description, location_lat, location_lng
		FROM events
		WHERE id = $1
	`
	e := &domain.Event{}
	var dateNull sql.NullTime
	var descNull sql.NullString
	var latNull, lngNull sql.NullFloat64
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&e.ID, &e.Name, &e.EventCode, &e.OwnerID, &e.CreatedAt, &e.UpdatedAt,
		&dateNull, &descNull, &latNull, &lngNull,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if dateNull.Valid {
		e.Date = &dateNull.Time
	}
	if descNull.Valid {
		e.Description = &descNull.String
	}
	if latNull.Valid {
		e.LocationLat = &latNull.Float64
	}
	if lngNull.Valid {
		e.LocationLng = &lngNull.Float64
	}
	return e, nil
}

func (r *eventRepository) GetByEventCode(ctx context.Context, eventCode string) (*domain.Event, error) {
	code := strings.ToLower(strings.TrimSpace(eventCode))
	query := `
		SELECT id, name, event_code, owner_id, created_at, updated_at, date, description, location_lat, location_lng
		FROM events
		WHERE event_code = $1
	`
	e := &domain.Event{}
	var dateNull sql.NullTime
	var descNull sql.NullString
	var latNull, lngNull sql.NullFloat64
	err := r.DB.QueryRowContext(ctx, query, code).Scan(
		&e.ID, &e.Name, &e.EventCode, &e.OwnerID, &e.CreatedAt, &e.UpdatedAt,
		&dateNull, &descNull, &latNull, &lngNull,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if dateNull.Valid {
		e.Date = &dateNull.Time
	}
	if descNull.Valid {
		e.Description = &descNull.String
	}
	if latNull.Valid {
		e.LocationLat = &latNull.Float64
	}
	if lngNull.Valid {
		e.LocationLng = &lngNull.Float64
	}
	return e, nil
}

func (r *eventRepository) ListByOwnerID(ctx context.Context, ownerID string) ([]*domain.Event, error) {
	query := `
		SELECT id, name, event_code, owner_id, created_at, updated_at, date, description, location_lat, location_lng
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
		var dateNull sql.NullTime
		var descNull sql.NullString
		var latNull, lngNull sql.NullFloat64
		if err := rows.Scan(&e.ID, &e.Name, &e.EventCode, &e.OwnerID, &e.CreatedAt, &e.UpdatedAt, &dateNull, &descNull, &latNull, &lngNull); err != nil {
			return nil, err
		}
		if dateNull.Valid {
			e.Date = &dateNull.Time
		}
		if descNull.Valid {
			e.Description = &descNull.String
		}
		if latNull.Valid {
			e.LocationLat = &latNull.Float64
		}
		if lngNull.Valid {
			e.LocationLng = &lngNull.Float64
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

func (r *eventRepository) Update(ctx context.Context, eventID string, date *time.Time, description *string, locationLat, locationLng *float64) (*domain.Event, error) {
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	n := 1
	if date != nil {
		setClauses = append(setClauses, fmt.Sprintf("date = $%d", n))
		args = append(args, *date)
		n++
	}
	if description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", n))
		args = append(args, *description)
		n++
	}
	if locationLat != nil {
		setClauses = append(setClauses, fmt.Sprintf("location_lat = $%d", n))
		args = append(args, *locationLat)
		n++
	}
	if locationLng != nil {
		setClauses = append(setClauses, fmt.Sprintf("location_lng = $%d", n))
		args = append(args, *locationLng)
		n++
	}
	if n == 1 {
		// No fields to update; just fetch current row
		return r.GetByID(ctx, eventID)
	}
	args = append(args, eventID)
	query := fmt.Sprintf(`
		UPDATE events SET %s
		WHERE id = $%d
		RETURNING id, name, event_code, owner_id, created_at, updated_at, date, description, location_lat, location_lng
	`, strings.Join(setClauses, ", "), n)
	e := &domain.Event{}
	var dateNull sql.NullTime
	var descNull sql.NullString
	var latNull, lngNull sql.NullFloat64
	err := r.DB.QueryRowContext(ctx, query, args...).Scan(
		&e.ID, &e.Name, &e.EventCode, &e.OwnerID, &e.CreatedAt, &e.UpdatedAt,
		&dateNull, &descNull, &latNull, &lngNull,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if dateNull.Valid {
		e.Date = &dateNull.Time
	}
	if descNull.Valid {
		e.Description = &descNull.String
	}
	if latNull.Valid {
		e.LocationLat = &latNull.Float64
	}
	if lngNull.Valid {
		e.LocationLng = &lngNull.Float64
	}
	return e, nil
}
