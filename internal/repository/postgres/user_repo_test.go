package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"multitrackticketing/internal/domain"

	"github.com/stretchr/testify/require"
)

func TestUserRepository_Update(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		user    *domain.User
		mock    func(mock sqlmock.Sqlmock)
		wantErr bool
		errIs   error
	}{
		{
			name: "success",
			user: &domain.User{
				ID:        "user-uuid-1",
				Email:     "alice@example.com",
				Name:      "Alice",
				UpdatedAt: time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC),
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WithArgs("Alice", "", "alice@example.com", time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC), "user-uuid-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "not found zero rows affected",
			user: &domain.User{
				ID:        "nonexistent",
				Email:     "a@b.com",
				Name:      "A",
				UpdatedAt: time.Now(),
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WithArgs("A", "", "a@b.com", sqlmock.AnyArg(), "nonexistent").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
			errIs:   domain.ErrUserNotFound,
		},
		{
			name: "unique violation returns ErrDuplicateEmail",
			user: &domain.User{
				ID:        "user-uuid-1",
				Email:     "taken@example.com",
				Name:      "Alice",
				UpdatedAt: time.Now(),
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WillReturnError(&pq.Error{Code: "23505"})
			},
			wantErr: true,
			errIs:   domain.ErrDuplicateEmail,
		},
		{
			name: "db error",
			user: &domain.User{
				ID:        "user-1",
				Email:     "a@b.com",
				Name:      "A",
				UpdatedAt: time.Now(),
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mock(mock)
			repo := NewUserRepository(db)
			err = repo.Update(ctx, tt.user)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs)
				}
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
