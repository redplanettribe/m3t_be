package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"multitrackticketing/internal/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestEventRepository_Create(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		event   *domain.Event
		mock    func(mock sqlmock.Sqlmock)
		wantID  string
		wantErr bool
	}{
		{
			name: "success",
			event: &domain.Event{
				Name:      "Conf 2025",
				EventCode: "ABCD",
				OwnerID:   "user-uuid-1",
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO events \(name, event_code, owner_id, created_at, updated_at\)`).
					WithArgs("Conf 2025", "ABCD", "user-uuid-1", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ev-uuid-1"))
			},
			wantID:  "ev-uuid-1",
			wantErr: false,
		},
		{
			name: "db error",
			event: &domain.Event{
				Name:      "Conf",
				EventCode: "WXYZ",
				OwnerID:   "user-1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO events`).
					WillReturnError(sql.ErrConnDone)
			},
			wantID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mock(mock)
			repo := NewEventRepository(db)
			err = repo.Create(ctx, tt.event)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantID, tt.event.ID)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEventRepository_GetByID(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cols := []string{"id", "name", "event_code", "owner_id", "created_at", "updated_at", "date", "description", "location_lat", "location_lng"}

	tests := []struct {
		name    string
		id      string
		mock    func(mock sqlmock.Sqlmock)
		want    *domain.Event
		wantErr bool
	}{
		{
			name: "success",
			id:   "ev-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at, date, description, location_lat, location_lng`).
					WithArgs("ev-1").
					WillReturnRows(sqlmock.NewRows(cols).
						AddRow("ev-1", "Conf", "ABCD", "user-1", createdAt, updatedAt, nil, nil, nil, nil))
			},
			want: &domain.Event{
				ID:        "ev-1",
				Name:      "Conf",
				EventCode: "ABCD",
				OwnerID:   "user-1",
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   "ev-missing",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at, date, description, location_lat, location_lng`).
					WithArgs("ev-missing").
					WillReturnError(sql.ErrNoRows)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mock(mock)
			repo := NewEventRepository(db)
			got, err := repo.GetByID(ctx, tt.id)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				require.NoError(t, mock.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEventRepository_GetByEventCode(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cols := []string{"id", "name", "event_code", "owner_id", "created_at", "updated_at", "date", "description", "location_lat", "location_lng"}

	tests := []struct {
		name      string
		eventCode string
		mock      func(mock sqlmock.Sqlmock)
		want      *domain.Event
		wantErr   bool
		isNotFound bool
	}{
		{
			name:      "success",
			eventCode: "abcd",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at, date, description, location_lat, location_lng`).
					WithArgs("abcd").
					WillReturnRows(sqlmock.NewRows(cols).
						AddRow("ev-1", "Conf", "abcd", "user-1", createdAt, updatedAt, nil, nil, nil, nil))
			},
			want: &domain.Event{
				ID:        "ev-1",
				Name:      "Conf",
				EventCode: "abcd",
				OwnerID:   "user-1",
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			wantErr: false,
		},
		{
			name:      "success normalizes to lowercase",
			eventCode: "ABCD",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at, date, description, location_lat, location_lng`).
					WithArgs("abcd").
					WillReturnRows(sqlmock.NewRows(cols).
						AddRow("ev-1", "Conf", "abcd", "user-1", createdAt, updatedAt, nil, nil, nil, nil))
			},
			want: &domain.Event{
				ID:        "ev-1",
				Name:      "Conf",
				EventCode: "abcd",
				OwnerID:   "user-1",
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			wantErr: false,
		},
		{
			name:      "not found",
			eventCode: "none",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at, date, description, location_lat, location_lng`).
					WithArgs("none").
					WillReturnError(sql.ErrNoRows)
			},
			want:       nil,
			wantErr:    true,
			isNotFound: true,
		},
		{
			name:      "db error",
			eventCode: "abcd",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at, date, description, location_lat, location_lng`).
					WithArgs("abcd").
					WillReturnError(sql.ErrConnDone)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mock(mock)
			repo := NewEventRepository(db)
			got, err := repo.GetByEventCode(ctx, tt.eventCode)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				if tt.isNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				require.NoError(t, mock.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEventRepository_ListByOwnerID(t *testing.T) {
	ctx := context.Background()
	createdAt1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	createdAt2 := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	updatedAt2 := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	cols := []string{"id", "name", "event_code", "owner_id", "created_at", "updated_at", "date", "description", "location_lat", "location_lng"}

	tests := []struct {
		name    string
		ownerID string
		mock    func(mock sqlmock.Sqlmock)
		want    []*domain.Event
		wantErr bool
	}{
		{
			name:    "success multiple",
			ownerID: "user-1",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(cols).
					AddRow("ev-1", "Conf A", "ABCD", "user-1", createdAt1, updatedAt1, nil, nil, nil, nil).
					AddRow("ev-2", "Conf B", "WXYZ", "user-1", createdAt2, updatedAt2, nil, nil, nil, nil)
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at, date, description, location_lat, location_lng`).
					WithArgs("user-1").
					WillReturnRows(rows)
			},
			want: []*domain.Event{
				{ID: "ev-1", Name: "Conf A", EventCode: "ABCD", OwnerID: "user-1", CreatedAt: createdAt1, UpdatedAt: updatedAt1},
				{ID: "ev-2", Name: "Conf B", EventCode: "WXYZ", OwnerID: "user-1", CreatedAt: createdAt2, UpdatedAt: updatedAt2},
			},
			wantErr: false,
		},
		{
			name:    "success empty",
			ownerID: "user-none",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at, date, description, location_lat, location_lng`).
					WithArgs("user-none").
					WillReturnRows(sqlmock.NewRows(cols))
			},
			want:    []*domain.Event{},
			wantErr: false,
		},
		{
			name:    "db error",
			ownerID: "user-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at, date, description, location_lat, location_lng`).
					WithArgs("user-1").
					WillReturnError(sql.ErrConnDone)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mock(mock)
			repo := NewEventRepository(db)
			got, err := repo.ListByOwnerID(ctx, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				require.NoError(t, mock.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEventRepository_Delete(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		id         string
		mock       func(mock sqlmock.Sqlmock)
		wantErr    bool
		isNotFound bool
	}{
		{
			name: "success",
			id:   "ev-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM events WHERE id = \$1`).
					WithArgs("ev-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr:     false,
			isNotFound: false,
		},
		{
			name: "not found",
			id:   "ev-missing",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM events WHERE id = \$1`).
					WithArgs("ev-missing").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr:     true,
			isNotFound: true,
		},
		{
			name: "db error",
			id:   "ev-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM events WHERE id = \$1`).
					WithArgs("ev-1").
					WillReturnError(sql.ErrConnDone)
			},
			wantErr:     true,
			isNotFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mock(mock)
			repo := NewEventRepository(db)
			err = repo.Delete(ctx, tt.id)
			if tt.wantErr {
				require.Error(t, err)
				if tt.isNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				require.NoError(t, mock.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEventRepository_Update(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	eventDate := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	desc := "Annual conf"
	lat, lng := 40.7128, -74.0060
	cols := []string{"id", "name", "event_code", "owner_id", "created_at", "updated_at", "date", "description", "location_lat", "location_lng"}

	tests := []struct {
		name        string
		eventID     string
		date        *time.Time
		description *string
		locationLat *float64
		locationLng *float64
		mock        func(mock sqlmock.Sqlmock)
		want        *domain.Event
		wantErr     bool
		isNotFound  bool
	}{
		{
			name:        "update date only",
			eventID:     "ev-1",
			date:        &eventDate,
			description: nil,
			locationLat: nil,
			locationLng: nil,
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE events SET updated_at = NOW\(\), date = \$1`).
					WithArgs(eventDate, "ev-1").
					WillReturnRows(sqlmock.NewRows(cols).
						AddRow("ev-1", "Conf", "ABCD", "user-1", createdAt, updatedAt, eventDate, nil, nil, nil))
			},
			want: &domain.Event{
				ID:        "ev-1",
				Name:      "Conf",
				EventCode: "ABCD",
				OwnerID:   "user-1",
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				Date:      &eventDate,
			},
			wantErr: false,
		},
		{
			name:        "update description only",
			eventID:     "ev-1",
			date:        nil,
			description: &desc,
			locationLat: nil,
			locationLng: nil,
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE events SET updated_at = NOW\(\), description = \$1`).
					WithArgs("Annual conf", "ev-1").
					WillReturnRows(sqlmock.NewRows(cols).
						AddRow("ev-1", "Conf", "ABCD", "user-1", createdAt, updatedAt, nil, desc, nil, nil))
			},
			want: &domain.Event{
				ID:          "ev-1",
				Name:        "Conf",
				EventCode:   "ABCD",
				OwnerID:     "user-1",
				CreatedAt:   createdAt,
				UpdatedAt:   updatedAt,
				Description: &desc,
			},
			wantErr: false,
		},
		{
			name:        "update location only",
			eventID:     "ev-1",
			date:        nil,
			description: nil,
			locationLat: &lat,
			locationLng: &lng,
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE events SET updated_at = NOW\(\), location_lat = \$1, location_lng = \$2`).
					WithArgs(40.7128, -74.006, "ev-1").
					WillReturnRows(sqlmock.NewRows(cols).
						AddRow("ev-1", "Conf", "ABCD", "user-1", createdAt, updatedAt, nil, nil, 40.7128, -74.006))
			},
			want: &domain.Event{
				ID:          "ev-1",
				Name:        "Conf",
				EventCode:   "ABCD",
				OwnerID:     "user-1",
				CreatedAt:   createdAt,
				UpdatedAt:   updatedAt,
				LocationLat: &lat,
				LocationLng: &lng,
			},
			wantErr: false,
		},
		{
			name:        "no fields to update calls GetByID",
			eventID:     "ev-1",
			date:        nil,
			description: nil,
			locationLat: nil,
			locationLng: nil,
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at, date, description, location_lat, location_lng`).
					WithArgs("ev-1").
					WillReturnRows(sqlmock.NewRows(cols).
						AddRow("ev-1", "Conf", "ABCD", "user-1", createdAt, updatedAt, nil, nil, nil, nil))
			},
			want: &domain.Event{
				ID:        "ev-1",
				Name:      "Conf",
				EventCode: "ABCD",
				OwnerID:   "user-1",
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			wantErr: false,
		},
		{
			name:        "not found",
			eventID:     "ev-missing",
			date:        &eventDate,
			description: nil,
			locationLat: nil,
			locationLng: nil,
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE events SET`).
					WithArgs(eventDate, "ev-missing").
					WillReturnError(sql.ErrNoRows)
			},
			want:       nil,
			wantErr:    true,
			isNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mock(mock)
			repo := NewEventRepository(db)
			got, err := repo.Update(ctx, tt.eventID, tt.date, tt.description, tt.locationLat, tt.locationLng)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				if tt.isNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				require.NoError(t, mock.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
