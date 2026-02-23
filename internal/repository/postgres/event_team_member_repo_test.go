package postgres

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"multitrackticketing/internal/domain"
)

func TestEventTeamMemberRepository_Add(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		eventID string
		userID  string
		mock    func(mock sqlmock.Sqlmock)
		wantErr error
	}{
		{
			name:    "success",
			eventID: "ev-1",
			userID:  "user-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO event_team_members \(event_id, user_id\)`).
					WithArgs("ev-1", "user-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: nil,
		},
		{
			name:    "duplicate returns ErrAlreadyMember",
			eventID: "ev-1",
			userID:  "user-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO event_team_members \(event_id, user_id\)`).
					WithArgs("ev-1", "user-1").
					WillReturnError(&pq.Error{Code: "23505"})
			},
			wantErr: domain.ErrAlreadyMember,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mock(mock)
			repo := NewEventTeamMemberRepository(db)
			err = repo.Add(ctx, tt.eventID, tt.userID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEventTeamMemberRepository_ListByEventID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		eventID string
		mock    func(mock sqlmock.Sqlmock)
		want    []*domain.EventTeamMember
		wantErr bool
	}{
		{
			name:    "success returns members",
			eventID: "ev-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT e.event_id, e.user_id, u.name, u.last_name, u.email`).
					WithArgs("ev-1").
					WillReturnRows(sqlmock.NewRows([]string{"event_id", "user_id", "name", "last_name", "email"}).
						AddRow("ev-1", "user-a", "Alice", "A", "alice@example.com").
						AddRow("ev-1", "user-b", "Bob", "B", "bob@example.com"))
			},
			want: []*domain.EventTeamMember{
				{EventID: "ev-1", UserID: "user-a", Name: "Alice", LastName: "A", Email: "alice@example.com"},
				{EventID: "ev-1", UserID: "user-b", Name: "Bob", LastName: "B", Email: "bob@example.com"},
			},
			wantErr: false,
		},
		{
			name:    "success empty",
			eventID: "ev-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT e.event_id, e.user_id, u.name, u.last_name, u.email`).
					WithArgs("ev-1").
					WillReturnRows(sqlmock.NewRows([]string{"event_id", "user_id", "name", "last_name", "email"}))
			},
			want:    []*domain.EventTeamMember{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mock(mock)
			repo := NewEventTeamMemberRepository(db)
			got, err := repo.ListByEventID(ctx, tt.eventID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEventTeamMemberRepository_Remove(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		eventID string
		userID  string
		mock    func(mock sqlmock.Sqlmock)
		wantErr error
	}{
		{
			name:    "success",
			eventID: "ev-1",
			userID:  "user-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM event_team_members WHERE event_id = \$1 AND user_id = \$2`).
					WithArgs("ev-1", "user-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: nil,
		},
		{
			name:    "no row returns ErrNotFound",
			eventID: "ev-1",
			userID:  "user-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM event_team_members WHERE event_id = \$1 AND user_id = \$2`).
					WithArgs("ev-1", "user-1").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: domain.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.mock(mock)
			repo := NewEventTeamMemberRepository(db)
			err = repo.Remove(ctx, tt.eventID, tt.userID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
