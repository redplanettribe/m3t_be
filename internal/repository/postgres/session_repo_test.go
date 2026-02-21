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

func TestSessionRepository_CreateRoom(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		room    *domain.Room
		mock    func(mock sqlmock.Sqlmock)
		wantID  string
		wantErr bool
	}{
		{
			name: "success",
			room: &domain.Room{
				EventID:          "ev-1",
				Name:             "Room A",
				SessionizeRoomID: 1,
				CreatedAt:        createdAt,
				UpdatedAt:        updatedAt,
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO rooms`).
					WithArgs("ev-1", "Room A", 1, createdAt, updatedAt).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("room-uuid-1"))
			},
			wantID:  "room-uuid-1",
			wantErr: false,
		},
		{
			name: "db error",
			room: &domain.Room{
				EventID:          "ev-1",
				Name:             "Room B",
				SessionizeRoomID: 2,
				CreatedAt:        createdAt,
				UpdatedAt:        updatedAt,
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO rooms`).
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
			repo := NewSessionRepository(db)
			err = repo.CreateRoom(ctx, tt.room)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantID, tt.room.ID)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSessionRepository_CreateSession(t *testing.T) {
	ctx := context.Background()
	startTime := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 3, 1, 11, 0, 0, 0, time.UTC)
	createdAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		session *domain.Session
		mock    func(mock sqlmock.Sqlmock)
		wantID  string
		wantErr bool
	}{
		{
			name: "success",
			session: &domain.Session{
				RoomID:              "room-1",
				SessionizeSessionID: "sess-1",
				Title:               "Talk 1",
				StartTime:           startTime,
				EndTime:             endTime,
				Description:         "A talk",
				CreatedAt:           createdAt,
				UpdatedAt:           updatedAt,
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO sessions`).
					WithArgs("room-1", "sess-1", "Talk 1", startTime, endTime, "A talk", createdAt, updatedAt).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("session-uuid-1"))
			},
			wantID:  "session-uuid-1",
			wantErr: false,
		},
		{
			name: "db error",
			session: &domain.Session{
				RoomID:              "room-1",
				SessionizeSessionID: "sess-2",
				Title:               "Talk 2",
				StartTime:           startTime,
				EndTime:             endTime,
				Description:         "",
				CreatedAt:           createdAt,
				UpdatedAt:           updatedAt,
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO sessions`).
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
			repo := NewSessionRepository(db)
			err = repo.CreateSession(ctx, tt.session)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantID, tt.session.ID)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSessionRepository_DeleteScheduleByEventID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		eventID string
		mock    func(mock sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name:    "success",
			eventID: "ev-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM rooms WHERE event_id`).
					WithArgs("ev-1").
					WillReturnResult(sqlmock.NewResult(0, 3))
			},
			wantErr: false,
		},
		{
			name:    "db error",
			eventID: "ev-2",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM rooms WHERE event_id`).
					WithArgs("ev-2").
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
			repo := NewSessionRepository(db)
			err = repo.DeleteScheduleByEventID(ctx, tt.eventID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
