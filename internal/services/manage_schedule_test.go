package services

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
	return nil, domain.ErrNotFound
}

func (f *fakeEventRepo) GetBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	if e, ok := f.bySlug[slug]; ok {
		return e, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeEventRepo) ListByOwnerID(ctx context.Context, ownerID string) ([]*domain.Event, error) {
	var out []*domain.Event
	for _, e := range f.byID {
		if e.OwnerID == ownerID {
			out = append(out, e)
		}
	}
	// Sort by CreatedAt DESC to match repo
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].CreatedAt.After(out[i].CreatedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
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

func (f *fakeSessionRepo) ListRoomsByEventID(ctx context.Context, eventID string) ([]*domain.Room, error) {
	var out []*domain.Room
	for _, r := range f.rooms {
		if r.EventID == eventID {
			out = append(out, r)
		}
	}
	if out == nil {
		return []*domain.Room{}, nil
	}
	return out, nil
}

func (f *fakeSessionRepo) ListSessionsByEventID(ctx context.Context, eventID string) ([]*domain.Session, error) {
	roomIDs := make(map[string]bool)
	for _, r := range f.rooms {
		if r.EventID == eventID {
			roomIDs[r.ID] = true
		}
	}
	var out []*domain.Session
	for _, s := range f.sessions {
		if roomIDs[s.RoomID] {
			out = append(out, s)
		}
	}
	if out == nil {
		return []*domain.Session{}, nil
	}
	return out, nil
}

// fakeSessionizeFetcher returns fixed data or a configurable error.
type fakeSessionizeFetcher struct {
	data domain.SessionizeResponse
	err  error
}

func (f *fakeSessionizeFetcher) Fetch(ctx context.Context, sessionizeID string) (domain.SessionizeResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.data, nil
}

// defaultSessionizeData returns a minimal valid SessionizeResponse for tests.
func defaultSessionizeData() domain.SessionizeResponse {
	desc := "A talk"
	return domain.SessionizeResponse{
		{
			Date: "2025-03-01",
			Rooms: []domain.SessionizeRoom{
				{
					ID:   1,
					Name: "Room A",
					Sessions: []domain.SessionizeSession{
						{
							ID:          "s1",
							Title:       "Talk 1",
							Description: &desc,
							StartsAt:    time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC),
							EndsAt:      time.Date(2025, 3, 1, 11, 0, 0, 0, time.UTC),
							RoomID:      1,
							Categories: []domain.SessionizeCategory{
								{Name: "Tipo de sesión", CategoryItems: []domain.SessionizeCategoryItem{{Name: "Conferencia"}}},
								{Name: "Event tag", CategoryItems: []domain.SessionizeCategoryItem{{Name: "ai"}, {Name: "web"}}},
							},
						},
					},
				},
			},
		},
	}
}

func TestManageScheduleService_CreateEvent(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name    string
		setup   func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher)
		event   *domain.Event
		wantErr bool
		assert  func(t *testing.T, eventRepo *fakeEventRepo, event *domain.Event)
	}{
		{
			name: "success",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			event:   &domain.Event{Name: "Conf", Slug: "conf-2025", OwnerID: "user-1"},
			wantErr: false,
			assert: func(t *testing.T, eventRepo *fakeEventRepo, event *domain.Event) {
				require.NotEmpty(t, event.ID)
				assert.False(t, event.CreatedAt.IsZero())
				assert.False(t, event.UpdatedAt.IsZero())
				assert.Equal(t, "Conf", event.Name)
				assert.Equal(t, "conf-2025", event.Slug)
				assert.Equal(t, "user-1", event.OwnerID)
				got, ok := eventRepo.byID[event.ID]
				require.True(t, ok)
				assert.Equal(t, event.ID, got.ID)
				assert.Equal(t, "user-1", got.OwnerID)
			},
		},
		{
			name: "repo error",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				er.err = errors.New("db error")
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			event:   &domain.Event{Name: "Conf", Slug: "conf-2025", OwnerID: "user-1"},
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeEventRepo, _ *domain.Event) {},
		},
		{
			name: "missing owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			event:   &domain.Event{Name: "Conf", Slug: "conf-2025"},
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeEventRepo, _ *domain.Event) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewManageScheduleService(eventRepo, sessionRepo, fetcher, timeout)
			ev := &domain.Event{Name: tt.event.Name, Slug: tt.event.Slug, OwnerID: tt.event.OwnerID}
			err := svc.CreateEvent(ctx, ev)
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

func TestManageScheduleService_ImportSessionizeData(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name    string
		setup   func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher)
		eventID string
		sessID  string
		wantErr bool
		assert  func(t *testing.T, sessionRepo *fakeSessionRepo)
	}{
		{
			name: "success",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
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
				assert.ElementsMatch(t, []string{"Conferencia", "ai", "web"}, sessionRepo.sessions[0].Tags)
			},
		},
		{
			name: "fetcher error",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{err: errors.New("fetch failed")}
			},
			eventID: "ev-1",
			sessID:  "x",
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeSessionRepo) {},
		},
		{
			name: "DeleteScheduleByEventID error",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
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
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
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
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
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
			svc := NewManageScheduleService(eventRepo, sessionRepo, fetcher, timeout)
			err := svc.ImportSessionizeData(ctx, tt.eventID, tt.sessID)
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

func TestManageScheduleService_ListEventsByOwner(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name    string
		setup   func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher)
		ownerID string
		wantLen int
		assert  func(t *testing.T, events []*domain.Event)
	}{
		{
			name: "returns only owner events",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				// Create two events for user-1, one for user-2
				_ = er.Create(ctx, &domain.Event{Name: "E1", Slug: "e1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				_ = er.Create(ctx, &domain.Event{Name: "E2", Slug: "e2", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				_ = er.Create(ctx, &domain.Event{Name: "Other", Slug: "other", OwnerID: "user-2", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			ownerID: "user-1",
			wantLen: 2,
			assert: func(t *testing.T, events []*domain.Event) {
				for _, e := range events {
					assert.Equal(t, "user-1", e.OwnerID)
				}
			},
		},
		{
			name: "empty for unknown owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "E1", Slug: "e1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			ownerID: "user-none",
			wantLen: 0,
			assert:  func(t *testing.T, events []*domain.Event) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewManageScheduleService(eventRepo, sessionRepo, fetcher, timeout)
			events, err := svc.ListEventsByOwner(ctx, tt.ownerID)
			require.NoError(t, err)
			require.Len(t, events, tt.wantLen)
			tt.assert(t, events)
		})
	}
}

func TestManageScheduleService_GetEventByID(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name      string
		setup     func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher)
		eventID   string
		wantErr   bool
		wantNotFound bool
		assert    func(t *testing.T, event *domain.Event, rooms []*domain.Room, sessions []*domain.Session)
	}{
		{
			name: "success with rooms and sessions",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", Slug: "conf-2025", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				ev, _ := er.GetByID(ctx, "ev-1")
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: ev.ID, Name: "Room A"}}
				sr.sessions = []*domain.Session{{ID: "sess-1", RoomID: "room-1", Title: "Talk 1", Tags: []string{}}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID: "ev-1",
			wantErr: false,
			wantNotFound: false,
			assert: func(t *testing.T, event *domain.Event, rooms []*domain.Room, sessions []*domain.Session) {
				require.NotNil(t, event)
				assert.Equal(t, "ev-1", event.ID)
				assert.Equal(t, "Conf", event.Name)
				require.Len(t, rooms, 1)
				assert.Equal(t, "Room A", rooms[0].Name)
				require.Len(t, sessions, 1)
				assert.Equal(t, "Talk 1", sessions[0].Title)
			},
		},
		{
			name: "event not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID: "ev-missing",
			wantErr: true,
			wantNotFound: true,
			assert: func(t *testing.T, _ *domain.Event, _ []*domain.Room, _ []*domain.Session) {},
		},
		{
			name: "success empty rooms and sessions",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", Slug: "conf-2025", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID: "ev-1",
			wantErr: false,
			wantNotFound: false,
			assert: func(t *testing.T, event *domain.Event, rooms []*domain.Room, sessions []*domain.Session) {
				require.NotNil(t, event)
				assert.Equal(t, "ev-1", event.ID)
				require.NotNil(t, rooms)
				require.Len(t, rooms, 0)
				require.NotNil(t, sessions)
				require.Len(t, sessions, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewManageScheduleService(eventRepo, sessionRepo, fetcher, timeout)
			event, rooms, sessions, err := svc.GetEventByID(ctx, tt.eventID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound), "expected ErrNotFound")
				}
				return
			}
			require.NoError(t, err)
			tt.assert(t, event, rooms, sessions)
		})
	}
}
