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
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at`).
					WithArgs("ev-1").
					WillReturnRows(sqlmock.NewRows([]string{"id", "name", "event_code", "owner_id", "created_at", "updated_at"}).
						AddRow("ev-1", "Conf", "ABCD", "user-1", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)))
			},
			want: &domain.Event{
				ID:        "ev-1",
				Name:      "Conf",
				EventCode: "ABCD",
				OwnerID:   "user-1",
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   "ev-missing",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at`).
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

func TestEventRepository_ListByOwnerID(t *testing.T) {
	ctx := context.Background()

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
				rows := sqlmock.NewRows([]string{"id", "name", "event_code", "owner_id", "created_at", "updated_at"}).
					AddRow("ev-1", "Conf A", "ABCD", "user-1", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)).
					AddRow("ev-2", "Conf B", "WXYZ", "user-1", time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC))
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at`).
					WithArgs("user-1").
					WillReturnRows(rows)
			},
			want: []*domain.Event{
				{ID: "ev-1", Name: "Conf A", EventCode: "ABCD", OwnerID: "user-1", CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
				{ID: "ev-2", Name: "Conf B", EventCode: "WXYZ", OwnerID: "user-1", CreatedAt: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)},
			},
			wantErr: false,
		},
		{
			name:    "success empty",
			ownerID: "user-none",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at`).
					WithArgs("user-none").
					WillReturnRows(sqlmock.NewRows([]string{"id", "name", "event_code", "owner_id", "created_at", "updated_at"}))
			},
			want:    []*domain.Event{},
			wantErr: false,
		},
		{
			name:    "db error",
			ownerID: "user-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, event_code, owner_id, created_at, updated_at`).
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
