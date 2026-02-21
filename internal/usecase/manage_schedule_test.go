package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"multitrackticketing/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeEventRepo is an in-memory EventRepository for tests.
type fakeEventRepo struct {
	byID   map[string]*domain.Event
	bySlug map[string]*domain.Event
	nextID int
	err    error // if set, Create returns this error
}

func newFakeEventRepo() *fakeEventRepo {
	return &fakeEventRepo{
		byID:   make(map[string]*domain.Event),
		bySlug: make(map[string]*domain.Event),
		nextID: 1,
	}
}

func (f *fakeEventRepo) Create(ctx context.Context, e *domain.Event) error {
	if f.err != nil {
		return f.err
	}
	e.ID = fmt.Sprintf("ev-%d", f.nextID)
	f.nextID++
	f.byID[e.ID] = e
	f.bySlug[e.Slug] = e
	return nil
}

func (f *fakeEventRepo) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	if e, ok := f.byID[id]; ok {
		return e, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeEventRepo) GetBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	if e, ok := f.bySlug[slug]; ok {
		return e, nil
	}
	return nil, errors.New("not found")
}

// fakeSessionRepo is an in-memory SessionRepository for tests.
type fakeSessionRepo struct {
	rooms    []*domain.Room
	sessions []*domain.Session
	roomID   int
	sessID   int
	createRoomErr   error
	createSessionErr error
	deleteErr       error
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{
		rooms:    nil,
		sessions: nil,
		roomID:   1,
		sessID:   1,
	}
}

func (f *fakeSessionRepo) CreateRoom(ctx context.Context, r *domain.Room) error {
	if f.createRoomErr != nil {
		return f.createRoomErr
	}
	r.ID = fmt.Sprintf("room-%d", f.roomID)
	f.roomID++
	f.rooms = append(f.rooms, r)
	return nil
}

func (f *fakeSessionRepo) CreateSession(ctx context.Context, s *domain.Session) error {
	if f.createSessionErr != nil {
		return f.createSessionErr
	}
	s.ID = fmt.Sprintf("sess-%d", f.sessID)
	f.sessID++
	f.sessions = append(f.sessions, s)
	return nil
}

func (f *fakeSessionRepo) DeleteScheduleByEventID(ctx context.Context, eventID string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	roomIDsForEvent := make(map[string]bool)
	var rooms []*domain.Room
	for _, r := range f.rooms {
		if r.EventID == eventID {
			roomIDsForEvent[r.ID] = true
		} else {
			rooms = append(rooms, r)
		}
	}
	f.rooms = rooms
	var sessions []*domain.Session
	for _, s := range f.sessions {
		if !roomIDsForEvent[s.RoomID] {
			sessions = append(sessions, s)
		}
	}
	f.sessions = sessions
	return nil
}

// fakeSessionizeFetcher returns fixed data or a configurable error.
type fakeSessionizeFetcher struct {
	data SessionizeResponse
	err  error
}

func (f *fakeSessionizeFetcher) Fetch(ctx context.Context, sessionizeID string) (SessionizeResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.data, nil
}

// defaultSessionizeData returns a minimal valid SessionizeResponse for tests.
func defaultSessionizeData() SessionizeResponse {
	desc := "A talk"
	return SessionizeResponse{
		{
			Date: "2025-03-01",
			Rooms: []SessionizeRoom{
				{
					ID:   1,
					Name: "Room A",
					Sessions: []SessionizeSession{
						{
							ID:          "s1",
							Title:       "Talk 1",
							Description: &desc,
							StartsAt:    time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC),
							EndsAt:      time.Date(2025, 3, 1, 11, 0, 0, 0, time.UTC),
							RoomID:      1,
						},
					},
				},
			},
		},
	}
}

func TestManageScheduleUseCase_CreateEvent(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name    string
		setup   func() (domain.EventRepository, domain.SessionRepository, SessionizeFetcher)
		event   *domain.Event
		wantErr bool
		assert  func(t *testing.T, eventRepo *fakeEventRepo, event *domain.Event)
	}{
		{
			name: "success",
			setup: func() (domain.EventRepository, domain.SessionRepository, SessionizeFetcher) {
				er := newFakeEventRepo()
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			event:   &domain.Event{Name: "Conf", Slug: "conf-2025"},
			wantErr: false,
			assert: func(t *testing.T, eventRepo *fakeEventRepo, event *domain.Event) {
				require.NotEmpty(t, event.ID)
				assert.False(t, event.CreatedAt.IsZero())
				assert.False(t, event.UpdatedAt.IsZero())
				assert.Equal(t, "Conf", event.Name)
				assert.Equal(t, "conf-2025", event.Slug)
				got, ok := eventRepo.byID[event.ID]
				require.True(t, ok)
				assert.Equal(t, event.ID, got.ID)
			},
		},
		{
			name: "repo error",
			setup: func() (domain.EventRepository, domain.SessionRepository, SessionizeFetcher) {
				er := newFakeEventRepo()
				er.err = errors.New("db error")
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			event:   &domain.Event{Name: "Conf", Slug: "conf-2025"},
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeEventRepo, _ *domain.Event) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			uc := NewManageScheduleUseCase(eventRepo, sessionRepo, fetcher, timeout)
			ev := &domain.Event{Name: tt.event.Name, Slug: tt.event.Slug}
			err := uc.CreateEvent(ctx, ev)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			fe, ok := eventRepo.(*fakeEventRepo)
			require.True(t, ok)
			tt.assert(t, fe, ev)
		})
	}
}

func TestManageScheduleUseCase_ImportSessionizeData(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name    string
		setup   func() (domain.EventRepository, domain.SessionRepository, SessionizeFetcher)
		eventID string
		sessID  string
		wantErr bool
		assert  func(t *testing.T, sessionRepo *fakeSessionRepo)
	}{
		{
			name: "success",
			setup: func() (domain.EventRepository, domain.SessionRepository, SessionizeFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{data: defaultSessionizeData()}
			},
			eventID: "ev-1",
			sessID:  "abc123",
			wantErr: false,
			assert: func(t *testing.T, sessionRepo *fakeSessionRepo) {
				require.Len(t, sessionRepo.rooms, 1)
				assert.Equal(t, "Room A", sessionRepo.rooms[0].Name)
				require.Len(t, sessionRepo.sessions, 1)
				assert.Equal(t, "Talk 1", sessionRepo.sessions[0].Title)
			},
		},
		{
			name: "fetcher error",
			setup: func() (domain.EventRepository, domain.SessionRepository, SessionizeFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{err: errors.New("fetch failed")}
			},
			eventID: "ev-1",
			sessID:  "x",
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeSessionRepo) {},
		},
		{
			name: "DeleteScheduleByEventID error",
			setup: func() (domain.EventRepository, domain.SessionRepository, SessionizeFetcher) {
				sr := newFakeSessionRepo()
				sr.deleteErr = errors.New("delete failed")
				return newFakeEventRepo(), sr, &fakeSessionizeFetcher{data: defaultSessionizeData()}
			},
			eventID: "ev-1",
			sessID:  "x",
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeSessionRepo) {},
		},
		{
			name: "CreateRoom error",
			setup: func() (domain.EventRepository, domain.SessionRepository, SessionizeFetcher) {
				sr := newFakeSessionRepo()
				sr.createRoomErr = errors.New("create room failed")
				return newFakeEventRepo(), sr, &fakeSessionizeFetcher{data: defaultSessionizeData()}
			},
			eventID: "ev-1",
			sessID:  "x",
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeSessionRepo) {},
		},
		{
			name: "CreateSession error",
			setup: func() (domain.EventRepository, domain.SessionRepository, SessionizeFetcher) {
				sr := newFakeSessionRepo()
				sr.createSessionErr = errors.New("create session failed")
				return newFakeEventRepo(), sr, &fakeSessionizeFetcher{data: defaultSessionizeData()}
			},
			eventID: "ev-1",
			sessID:  "x",
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeSessionRepo) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			uc := NewManageScheduleUseCase(eventRepo, sessionRepo, fetcher, timeout)
			err := uc.ImportSessionizeData(ctx, tt.eventID, tt.sessID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			sr, ok := sessionRepo.(*fakeSessionRepo)
			require.True(t, ok)
			tt.assert(t, sr)
		})
	}
}