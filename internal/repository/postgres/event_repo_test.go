package postgres

import (
	"context"
	"database/sql"
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
				Slug:      "conf-2025",
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO events \(name, slug, created_at, updated_at\)`).
					WithArgs("Conf 2025", "conf-2025", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("ev-uuid-1"))
			},
			wantID:  "ev-uuid-1",
			wantErr: false,
		},
		{
			name: "db error",
			event: &domain.Event{
				Name:      "Conf",
				Slug:      "conf",
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
				mock.ExpectQuery(`SELECT id, name, slug, created_at, updated_at`).
					WithArgs("ev-1").
					WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "created_at", "updated_at"}).
						AddRow("ev-1", "Conf", "conf-2025", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)))
			},
			want: &domain.Event{
				ID:        "ev-1",
				Name:      "Conf",
				Slug:      "conf-2025",
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   "ev-missing",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, slug, created_at, updated_at`).
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

func TestEventRepository_GetBySlug(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		slug    string
		mock    func(mock sqlmock.Sqlmock)
		want    *domain.Event
		wantErr bool
	}{
		{
			name: "success",
			slug: "conf-2025",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, slug, created_at, updated_at`).
					WithArgs("conf-2025").
					WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "created_at", "updated_at"}).
						AddRow("ev-1", "Conf", "conf-2025", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)))
			},
			want: &domain.Event{
				ID:        "ev-1",
				Name:      "Conf",
				Slug:      "conf-2025",
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "not found",
			slug: "missing-slug",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name, slug, created_at, updated_at`).
					WithArgs("missing-slug").
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
			got, err := repo.GetBySlug(ctx, tt.slug)
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
