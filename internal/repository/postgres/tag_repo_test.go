package postgres

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestTagRepository_EnsureTagForEvent(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		eventID  string
		tagName  string
		mock     func(mock sqlmock.Sqlmock)
		wantID   string
		wantErr  bool
	}{
		{
			name:    "existing tag returns id and ensures event_tag",
			eventID: "ev-1",
			tagName: "ai",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id FROM tags WHERE name = \$1`).
					WithArgs("ai").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tag-uuid-1"))
				mock.ExpectExec(`INSERT INTO event_tags`).
					WithArgs("ev-1", "tag-uuid-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantID:  "tag-uuid-1",
			wantErr: false,
		},
		{
			name:    "new tag creates then ensures event_tag",
			eventID: "ev-2",
			tagName: "web",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id FROM tags WHERE name = \$1`).
					WithArgs("web").
					WillReturnError(sql.ErrNoRows)
				mock.ExpectQuery(`INSERT INTO tags`).
					WithArgs("web").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tag-uuid-2"))
				mock.ExpectExec(`INSERT INTO event_tags`).
					WithArgs("ev-2", "tag-uuid-2").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantID:  "tag-uuid-2",
			wantErr: false,
		},
		{
			name:    "event_tags idempotent on conflict",
			eventID: "ev-1",
			tagName: "ai",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id FROM tags WHERE name = \$1`).
					WithArgs("ai").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tag-uuid-1"))
				mock.ExpectExec(`INSERT INTO event_tags`).
					WithArgs("ev-1", "tag-uuid-1").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantID:  "tag-uuid-1",
			wantErr: false,
		},
		{
			name:    "select tag db error",
			eventID: "ev-1",
			tagName: "x",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id FROM tags WHERE name = \$1`).
					WithArgs("x").
					WillReturnError(sql.ErrConnDone)
			},
			wantID:  "",
			wantErr: true,
		},
		{
			name:    "insert tag db error",
			eventID: "ev-1",
			tagName: "y",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id FROM tags WHERE name = \$1`).
					WithArgs("y").
					WillReturnError(sql.ErrNoRows)
				mock.ExpectQuery(`INSERT INTO tags`).
					WithArgs("y").
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
			repo := NewTagRepository(db)
			got, err := repo.EnsureTagForEvent(ctx, tt.eventID, tt.tagName)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantID, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTagRepository_SetSessionTags(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		sessionID string
		tagIDs    []string
		mock      func(mock sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name:      "replace with two tags",
			sessionID: "sess-1",
			tagIDs:    []string{"tag-1", "tag-2"},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM session_tags WHERE session_id`).
					WithArgs("sess-1").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(`INSERT INTO session_tags`).WithArgs("sess-1", "tag-1").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`INSERT INTO session_tags`).WithArgs("sess-1", "tag-2").WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:      "replace with empty list clears tags",
			sessionID: "sess-2",
			tagIDs:    nil,
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM session_tags WHERE session_id`).
					WithArgs("sess-2").
					WillReturnResult(sqlmock.NewResult(0, 2))
			},
			wantErr: false,
		},
		{
			name:      "delete error",
			sessionID: "sess-1",
			tagIDs:    []string{"tag-1"},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM session_tags WHERE session_id`).
					WithArgs("sess-1").
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
			repo := NewTagRepository(db)
			err = repo.SetSessionTags(ctx, tt.sessionID, tt.tagIDs)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
