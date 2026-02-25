package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"multitrackticketing/internal/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
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
				NotBookable:      false,
				Capacity:         0,
				Description:      "",
				HowToGetThere:    "",
				CreatedAt:        createdAt,
				UpdatedAt:        updatedAt,
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO rooms`).
					WithArgs("ev-1", "Room A", 1, false, 0, "", "", createdAt, updatedAt).
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
				NotBookable:      false,
				Capacity:         0,
				Description:      "",
				HowToGetThere:    "",
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
				RoomID:          "room-1",
				SourceSessionID: "sess-1",
				Title:           "Talk 1",
				StartTime:       startTime,
				EndTime:         endTime,
				Description:     "A talk",
				Tags:            []string{},
				CreatedAt:       createdAt,
				UpdatedAt:       updatedAt,
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO sessions`).
					WithArgs("room-1", "sess-1", "Talk 1", startTime, endTime, "A talk", createdAt, updatedAt).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("session-uuid-1"))
				mock.ExpectExec(`DELETE FROM session_tags WHERE session_id`).
					WithArgs("session-uuid-1").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantID:  "session-uuid-1",
			wantErr: false,
		},
		{
			name: "success with tags",
			session: &domain.Session{
				RoomID:          "room-1",
				SourceSessionID: "sess-tags",
				Title:           "Talk with tags",
				StartTime:       startTime,
				EndTime:         endTime,
				Description:     "",
				Tags:            []string{"ai", "web"},
				CreatedAt:       createdAt,
				UpdatedAt:       updatedAt,
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO sessions`).
					WithArgs("room-1", "sess-tags", "Talk with tags", startTime, endTime, "", createdAt, updatedAt).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("session-uuid-2"))
				mock.ExpectExec(`DELETE FROM session_tags WHERE session_id`).
					WithArgs("session-uuid-2").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(`INSERT INTO session_tags`).WithArgs("session-uuid-2", "ai").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`INSERT INTO session_tags`).WithArgs("session-uuid-2", "web").WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantID:  "session-uuid-2",
			wantErr: false,
		},
		{
			name: "db error",
			session: &domain.Session{
				RoomID:          "room-1",
				SourceSessionID: "sess-2",
				Title:           "Talk 2",
				StartTime:       startTime,
				EndTime:         endTime,
				Description:     "",
				Tags:            nil,
				CreatedAt:       createdAt,
				UpdatedAt:       updatedAt,
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

func TestSessionRepository_ListRoomsByEventID(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		eventID string
		mock    func(mock sqlmock.Sqlmock)
		wantLen int
		wantErr bool
	}{
		{
			name:    "success two rooms",
			eventID: "ev-1",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "event_id", "name", "source_session_id", "not_bookable", "capacity", "description", "how_to_get_there", "created_at", "updated_at"}).
					AddRow("room-1", "ev-1", "Room A", 1, false, 0, "", "", createdAt, updatedAt).
					AddRow("room-2", "ev-1", "Room B", 2, true, 0, "", "", createdAt, updatedAt)
				mock.ExpectQuery(`SELECT id, event_id, name, source_session_id, not_bookable, capacity, description, how_to_get_there, created_at, updated_at`).
					WithArgs("ev-1").
					WillReturnRows(rows)
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "success empty",
			eventID: "ev-2",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, event_id, name, source_session_id, not_bookable, capacity, description, how_to_get_there, created_at, updated_at`).
					WithArgs("ev-2").
					WillReturnRows(sqlmock.NewRows([]string{"id", "event_id", "name", "source_session_id", "not_bookable", "capacity", "description", "how_to_get_there", "created_at", "updated_at"}))
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "db error",
			eventID: "ev-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, event_id, name, source_session_id, not_bookable, capacity, description, how_to_get_there, created_at, updated_at`).
					WithArgs("ev-1").
					WillReturnError(sql.ErrConnDone)
			},
			wantLen: 0,
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
			rooms, err := repo.ListRoomsByEventID(ctx, tt.eventID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, rooms, tt.wantLen)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSessionRepository_GetRoomByID(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		roomID  string
		mock    func(mock sqlmock.Sqlmock)
		want    *domain.Room
		wantErr bool
	}{
		{
			name:   "success",
			roomID: "room-1",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "event_id", "name", "source_session_id", "not_bookable", "capacity", "description", "how_to_get_there", "created_at", "updated_at"}).
					AddRow("room-1", "ev-1", "Room A", 1, false, 0, "", "", createdAt, updatedAt)
				mock.ExpectQuery(`SELECT id, event_id, name, source_session_id, not_bookable, capacity, description, how_to_get_there, created_at, updated_at`).
					WithArgs("room-1").
					WillReturnRows(rows)
			},
			want: &domain.Room{
				ID:               "room-1",
				EventID:          "ev-1",
				Name:             "Room A",
				SessionizeRoomID: 1,
				NotBookable:      false,
				Capacity:         0,
				Description:      "",
				HowToGetThere:    "",
				CreatedAt:        createdAt,
				UpdatedAt:        updatedAt,
			},
			wantErr: false,
		},
		{
			name:   "not found",
			roomID: "room-missing",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, event_id, name, source_session_id, not_bookable, capacity, description, how_to_get_there, created_at, updated_at`).
					WithArgs("room-missing").
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
			repo := NewSessionRepository(db)
			room, err := repo.GetRoomByID(ctx, tt.roomID)
			if tt.wantErr {
				require.Error(t, err)
				require.True(t, errors.Is(err, domain.ErrNotFound))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, room)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSessionRepository_SetRoomNotBookable(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		roomID       string
		notBookable  bool
		mock         func(mock sqlmock.Sqlmock)
		wantNotBook  bool
		wantErr      bool
		wantNotFound bool
	}{
		{
			name:        "success set true",
			roomID:      "room-1",
			notBookable: true,
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "event_id", "name", "source_session_id", "not_bookable", "capacity", "description", "how_to_get_there", "created_at", "updated_at"}).
					AddRow("room-1", "ev-1", "Room A", 1, true, 0, "", "", createdAt, updatedAt)
				mock.ExpectQuery(`UPDATE rooms`).
					WithArgs("room-1", true).
					WillReturnRows(rows)
			},
			wantNotBook:  true,
			wantErr:      false,
			wantNotFound: false,
		},
		{
			name:        "success set false",
			roomID:      "room-1",
			notBookable: false,
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "event_id", "name", "source_session_id", "not_bookable", "capacity", "description", "how_to_get_there", "created_at", "updated_at"}).
					AddRow("room-1", "ev-1", "Room A", 1, false, 0, "", "", createdAt, updatedAt)
				mock.ExpectQuery(`UPDATE rooms`).
					WithArgs("room-1", false).
					WillReturnRows(rows)
			},
			wantNotBook:  false,
			wantErr:      false,
			wantNotFound: false,
		},
		{
			name:        "not found",
			roomID:      "room-missing",
			notBookable: true,
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE rooms`).
					WithArgs("room-missing", true).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr:      true,
			wantNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			tt.mock(mock)
			repo := NewSessionRepository(db)
			room, err := repo.SetRoomNotBookable(ctx, tt.roomID, tt.notBookable)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, room)
			require.Equal(t, tt.wantNotBook, room.NotBookable)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSessionRepository_UpdateRoomDetails(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		roomID        string
		capacity      int
		description   string
		howToGetThere string
		notBookable   bool
		mock          func(mock sqlmock.Sqlmock)
		want          *domain.Room
		wantErr       bool
		wantNotFound  bool
	}{
		{
			name:          "success",
			roomID:        "room-1",
			capacity:      50,
			description:   "Main hall",
			howToGetThere: "Turn left at entrance",
			notBookable:   true,
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "event_id", "name", "source_session_id", "not_bookable", "capacity", "description", "how_to_get_there", "created_at", "updated_at"}).
					AddRow("room-1", "ev-1", "Room A", 1, true, 50, "Main hall", "Turn left at entrance", createdAt, updatedAt)
				mock.ExpectQuery(`UPDATE rooms`).
					WithArgs("room-1", 50, "Main hall", "Turn left at entrance", true).
					WillReturnRows(rows)
			},
			want: &domain.Room{
				ID:               "room-1",
				EventID:          "ev-1",
				Name:             "Room A",
				SessionizeRoomID: 1,
				NotBookable:      true,
				Capacity:         50,
				Description:      "Main hall",
				HowToGetThere:    "Turn left at entrance",
				CreatedAt:        createdAt,
				UpdatedAt:        updatedAt,
			},
			wantErr: false,
		},
		{
			name:          "not found",
			roomID:        "room-missing",
			capacity:      0,
			description:   "",
			howToGetThere: "",
			notBookable:   false,
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE rooms`).
					WithArgs("room-missing", 0, "", "", false).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name:          "db error",
			roomID:        "room-1",
			capacity:      10,
			description:   "",
			howToGetThere: "",
			notBookable:   false,
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE rooms`).
					WithArgs("room-1", 10, "", "", false).
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
			room, err := repo.UpdateRoomDetails(ctx, tt.roomID, tt.capacity, tt.description, tt.howToGetThere, tt.notBookable)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, room)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSessionRepository_DeleteRoom(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		roomID       string
		mock         func(mock sqlmock.Sqlmock)
		wantErr      bool
		wantNotFound bool
	}{
		{
			name:   "success",
			roomID: "room-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM rooms WHERE id`).
					WithArgs("room-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:   "not found",
			roomID: "room-missing",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM rooms WHERE id`).
					WithArgs("room-missing").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name:   "db error",
			roomID: "room-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM rooms WHERE id`).
					WithArgs("room-1").
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
			err = repo.DeleteRoom(ctx, tt.roomID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				return
			}
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSessionRepository_DeleteSession(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		sessionID    string
		mock         func(mock sqlmock.Sqlmock)
		wantErr      bool
		wantNotFound bool
	}{
		{
			name:      "success",
			sessionID: "sess-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM sessions WHERE id`).
					WithArgs("sess-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:      "not found",
			sessionID: "sess-missing",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM sessions WHERE id`).
					WithArgs("sess-missing").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name:      "db error",
			sessionID: "sess-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM sessions WHERE id`).
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
			repo := NewSessionRepository(db)
			err = repo.DeleteSession(ctx, tt.sessionID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				return
			}
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSessionRepository_ListSessionsByEventID(t *testing.T) {
	ctx := context.Background()
	startTime := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 3, 1, 11, 0, 0, 0, time.UTC)
	createdAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name                 string
		eventID              string
		mock                 func(mock sqlmock.Sqlmock)
		wantLen              int
		wantFirstSessionTags []string
		wantErr              bool
	}{
		{
			name:    "success one session",
			eventID: "ev-1",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "room_id", "sessionize_session_id", "title", "start_time", "end_time", "description", "created_at", "updated_at"}).
					AddRow("sess-1", "room-1", "s1", "Talk 1", startTime, endTime, "Desc", createdAt, updatedAt)
				mock.ExpectQuery(`SELECT s.id, s.room_id, s.sessionize_session_id, s.title, s.start_time, s.end_time, s.description, s.created_at, s.updated_at`).
					WithArgs("ev-1").
					WillReturnRows(rows)
				tagRows := sqlmock.NewRows([]string{"session_id", "tag"}).
					AddRow("sess-1", "ai").
					AddRow("sess-1", "web")
				mock.ExpectQuery(`SELECT session_id, tag FROM session_tags WHERE session_id = ANY`).
					WithArgs(pq.Array([]string{"sess-1"})).
					WillReturnRows(tagRows)
			},
			wantLen:              1,
			wantFirstSessionTags: []string{"ai", "web"},
			wantErr:              false,
		},
		{
			name:    "success empty",
			eventID: "ev-2",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT s.id, s.room_id, s.sessionize_session_id, s.title, s.start_time, s.end_time, s.description, s.created_at, s.updated_at`).
					WithArgs("ev-2").
					WillReturnRows(sqlmock.NewRows([]string{"id", "room_id", "sessionize_session_id", "title", "start_time", "end_time", "description", "created_at", "updated_at"}))
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "db error",
			eventID: "ev-1",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT s.id, s.room_id, s.sessionize_session_id, s.title, s.start_time, s.end_time, s.description, s.created_at, s.updated_at`).
					WithArgs("ev-1").
					WillReturnError(sql.ErrConnDone)
			},
			wantLen: 0,
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
			sessions, err := repo.ListSessionsByEventID(ctx, tt.eventID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, sessions, tt.wantLen)
			if tt.wantFirstSessionTags != nil && len(sessions) > 0 {
				require.ElementsMatch(t, tt.wantFirstSessionTags, sessions[0].Tags)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
