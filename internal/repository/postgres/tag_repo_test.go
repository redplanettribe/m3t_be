package postgres

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"multitrackticketing/internal/domain"
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

func TestTagRepository_AddSessionTag(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		sessionID string
		tagID     string
		mock      func(mock sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name:      "success",
			sessionID: "sess-1",
			tagID:     "tag-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO session_tags`).
					WithArgs("sess-1", "tag-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:      "idempotent no op",
			sessionID: "sess-2",
			tagID:     "tag-2",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO session_tags`).
					WithArgs("sess-2", "tag-2").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name:      "db error",
			sessionID: "sess-1",
			tagID:     "tag-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO session_tags`).
					WithArgs("sess-1", "tag-1").
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
			err = repo.AddSessionTag(ctx, tt.sessionID, tt.tagID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTagRepository_RemoveSessionTag(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		sessionID string
		tagID     string
		mock      func(mock sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name:      "success",
			sessionID: "sess-1",
			tagID:     "tag-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM session_tags WHERE session_id = \$1 AND tag_id = \$2`).
					WithArgs("sess-1", "tag-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:      "no rows affected still success",
			sessionID: "sess-2",
			tagID:     "tag-2",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM session_tags WHERE session_id = \$1 AND tag_id = \$2`).
					WithArgs("sess-2", "tag-2").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name:      "db error",
			sessionID: "sess-1",
			tagID:     "tag-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM session_tags`).
					WithArgs("sess-1", "tag-1").
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
			err = repo.RemoveSessionTag(ctx, tt.sessionID, tt.tagID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTagRepository_RemoveEventTag(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		eventID string
		tagID   string
		mock    func(mock sqlmock.Sqlmock)
		wantErr bool
		errIs   error
	}{
		{
			name:    "success deletes session_tags then event_tags",
			eventID: "ev-1",
			tagID:   "tag-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM session_tags WHERE tag_id = \$1 AND session_id IN`).
					WithArgs("tag-1", "ev-1").
					WillReturnResult(sqlmock.NewResult(0, 2))
				mock.ExpectExec(`DELETE FROM event_tags WHERE event_id = \$1 AND tag_id = \$2`).
					WithArgs("ev-1", "tag-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:    "tag not on event returns ErrNotFound",
			eventID: "ev-1",
			tagID:   "tag-999",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM session_tags WHERE tag_id = \$1 AND session_id IN`).
					WithArgs("tag-999", "ev-1").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(`DELETE FROM event_tags WHERE event_id = \$1 AND tag_id = \$2`).
					WithArgs("ev-1", "tag-999").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
			errIs:   domain.ErrNotFound,
		},
		{
			name:    "first delete db error",
			eventID: "ev-1",
			tagID:   "tag-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM session_tags WHERE tag_id = \$1 AND session_id IN`).
					WithArgs("tag-1", "ev-1").
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
			err = repo.RemoveEventTag(ctx, tt.eventID, tt.tagID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTagRepository_UpdateTagName(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		tagID   string
		newName string
		mock    func(mock sqlmock.Sqlmock)
		wantErr bool
		errIs   error
	}{
		{
			name:    "success",
			tagID:   "tag-1",
			newName: "NewName",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE tags SET name = \$2 WHERE id = \$1`).
					WithArgs("tag-1", "NewName").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:    "not found",
			tagID:   "tag-missing",
			newName: "X",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE tags SET name = \$2 WHERE id = \$1`).
					WithArgs("tag-missing", "X").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
			errIs:   domain.ErrNotFound,
		},
		{
			name:    "db error",
			tagID:   "tag-1",
			newName: "Y",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE tags SET name = \$2 WHERE id = \$1`).
					WithArgs("tag-1", "Y").
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
			err = repo.UpdateTagName(ctx, tt.tagID, tt.newName)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTagRepository_GetTagByID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		tagID   string
		mock    func(mock sqlmock.Sqlmock)
		wantTag *domain.Tag
		wantErr bool
		errIs   error
	}{
		{
			name:  "success",
			tagID: "tag-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name FROM tags WHERE id = \$1`).
					WithArgs("tag-1").
					WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("tag-1", "Go"))
			},
			wantTag: &domain.Tag{ID: "tag-1", Name: "Go"},
			wantErr: false,
		},
		{
			name:  "not found",
			tagID: "tag-missing",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name FROM tags WHERE id = \$1`).
					WithArgs("tag-missing").
					WillReturnError(sql.ErrNoRows)
			},
			wantTag: nil,
			wantErr: true,
			errIs:   domain.ErrNotFound,
		},
		{
			name:  "db error",
			tagID: "tag-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, name FROM tags WHERE id = \$1`).
					WithArgs("tag-1").
					WillReturnError(sql.ErrConnDone)
			},
			wantTag: nil,
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
			got, err := repo.GetTagByID(ctx, tt.tagID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantTag, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
