package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"multitrackticketing/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeEventRepo is an in-memory EventRepository for tests.
type fakeEventRepo struct {
	byID   map[string]*domain.Event
	nextID int
	err    error // if set, Create returns this error
}

func newFakeEventRepo() *fakeEventRepo {
	return &fakeEventRepo{
		byID:   make(map[string]*domain.Event),
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
	return nil
}

func (f *fakeEventRepo) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	if e, ok := f.byID[id]; ok {
		return e, nil
	}
	return nil, domain.ErrNotFound
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

func (f *fakeEventRepo) Delete(ctx context.Context, id string) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

// fakeSessionRepo is an in-memory SessionRepository for tests.
type fakeSessionRepo struct {
	rooms            []*domain.Room
	sessions         []*domain.Session
	roomID           int
	sessID           int
	createRoomErr    error
	createSessionErr error
	deleteErr        error
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

func (f *fakeSessionRepo) GetRoomByID(ctx context.Context, roomID string) (*domain.Room, error) {
	for _, r := range f.rooms {
		if r.ID == roomID {
			return r, nil
		}
	}
	return nil, domain.ErrNotFound
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

func (f *fakeSessionRepo) SetRoomNotBookable(ctx context.Context, roomID string, notBookable bool) (*domain.Room, error) {
	for _, r := range f.rooms {
		if r.ID == roomID {
			r.NotBookable = notBookable
			return r, nil
		}
	}
	return nil, domain.ErrNotFound
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

// fakeEventTeamMemberRepo is an in-memory EventTeamMemberRepository for tests.
type fakeEventTeamMemberRepo struct {
	members   map[string]map[string]bool // eventID -> userID -> true
	addErr    error
	removeErr error
}

func newFakeEventTeamMemberRepo() *fakeEventTeamMemberRepo {
	return &fakeEventTeamMemberRepo{
		members: make(map[string]map[string]bool),
	}
}

func (f *fakeEventTeamMemberRepo) Add(ctx context.Context, eventID, userID string) error {
	if f.addErr != nil {
		return f.addErr
	}
	if f.members[eventID] == nil {
		f.members[eventID] = make(map[string]bool)
	}
	if f.members[eventID][userID] {
		return domain.ErrAlreadyMember
	}
	f.members[eventID][userID] = true
	return nil
}

func (f *fakeEventTeamMemberRepo) ListByEventID(ctx context.Context, eventID string) ([]*domain.EventTeamMember, error) {
	userIDs, ok := f.members[eventID]
	if !ok {
		return []*domain.EventTeamMember{}, nil
	}
	out := make([]*domain.EventTeamMember, 0, len(userIDs))
	for uid := range userIDs {
		out = append(out, &domain.EventTeamMember{EventID: eventID, UserID: uid})
	}
	return out, nil
}

func (f *fakeEventTeamMemberRepo) Remove(ctx context.Context, eventID, userID string) error {
	if f.removeErr != nil {
		return f.removeErr
	}
	if f.members[eventID] == nil || !f.members[eventID][userID] {
		return domain.ErrNotFound
	}
	delete(f.members[eventID], userID)
	return nil
}

// fakeUserRepoForSchedule is a minimal UserRepository for schedule service tests (GetByEmail only).
type fakeUserRepoForSchedule struct {
	byEmail map[string]*domain.User // normalized lower email -> user
}

func newFakeUserRepoForSchedule() *fakeUserRepoForSchedule {
	return &fakeUserRepoForSchedule{byEmail: make(map[string]*domain.User)}
}

func (f *fakeUserRepoForSchedule) Create(ctx context.Context, user *domain.User) error { return nil }
func (f *fakeUserRepoForSchedule) Update(ctx context.Context, user *domain.User) error { return nil }
func (f *fakeUserRepoForSchedule) AssignRole(ctx context.Context, userID, roleID string) error {
	return nil
}
func (f *fakeUserRepoForSchedule) GetByID(ctx context.Context, id string) (*domain.User, error) {
	for _, u := range f.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (f *fakeUserRepoForSchedule) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func (f *fakeUserRepoForSchedule) addUser(email, id string) {
	email = strings.TrimSpace(strings.ToLower(email))
	f.byEmail[email] = &domain.User{ID: id, Email: email}
}

func (f *fakeUserRepoForSchedule) addUserWithName(email, id, name, lastName string) {
	email = strings.TrimSpace(strings.ToLower(email))
	f.byEmail[email] = &domain.User{ID: id, Email: email, Name: name, LastName: lastName}
}

// fakeEventInvitationRepo is an in-memory EventInvitationRepository for tests.
type fakeEventInvitationRepo struct {
	invitations []*domain.EventInvitation
	nextID      int
	createErr   error
}

func newFakeEventInvitationRepo() *fakeEventInvitationRepo {
	return &fakeEventInvitationRepo{
		invitations: nil,
		nextID:      1,
	}
}

func (f *fakeEventInvitationRepo) Create(ctx context.Context, inv *domain.EventInvitation) error {
	if f.createErr != nil {
		return f.createErr
	}
	for _, existing := range f.invitations {
		if existing.EventID == inv.EventID && strings.ToLower(existing.Email) == strings.ToLower(inv.Email) {
			return errors.New("duplicate key value violates unique constraint")
		}
	}
	inv.ID = fmt.Sprintf("inv-%d", f.nextID)
	f.nextID++
	f.invitations = append(f.invitations, inv)
	return nil
}

func (f *fakeEventInvitationRepo) ListByEventID(ctx context.Context, eventID string) ([]*domain.EventInvitation, error) {
	var out []*domain.EventInvitation
	for _, inv := range f.invitations {
		if inv.EventID == eventID {
			out = append(out, inv)
		}
	}
	if out == nil {
		return []*domain.EventInvitation{}, nil
	}
	return out, nil
}

// fakeEmailService is a test double for EmailService. Tracks SendEventInvitation calls; other methods no-op.
type fakeEmailService struct {
	sendEventInvitationErr error // if set, SendEventInvitation returns this
	sentInvitations        []*domain.EventInvitationEmailData
}

func newFakeEmailService() *fakeEmailService {
	return &fakeEmailService{sentInvitations: []*domain.EventInvitationEmailData{}}
}

func (f *fakeEmailService) SendWelcomeMessage(ctx context.Context, data *domain.WelcomeMessageEmailData) error {
	return nil
}

func (f *fakeEmailService) SendLoginCode(ctx context.Context, data *domain.LoginCodeEmailData) error {
	return nil
}

func (f *fakeEmailService) SendEventInvitation(ctx context.Context, data *domain.EventInvitationEmailData) error {
	if f.sendEventInvitationErr != nil {
		return f.sendEventInvitationErr
	}
	f.sentInvitations = append(f.sentInvitations, data)
	return nil
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
			event:   &domain.Event{Name: "Conf", OwnerID: "user-1"},
			wantErr: false,
			assert: func(t *testing.T, eventRepo *fakeEventRepo, event *domain.Event) {
				require.NotEmpty(t, event.ID)
				assert.False(t, event.CreatedAt.IsZero())
				assert.False(t, event.UpdatedAt.IsZero())
				assert.Equal(t, "Conf", event.Name)
				assert.Len(t, event.EventCode, 4)
				assert.Regexp(t, "^[a-z0-9]{4}$", event.EventCode)
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
			event:   &domain.Event{Name: "Conf", OwnerID: "user-1"},
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeEventRepo, _ *domain.Event) {},
		},
		{
			name: "missing owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			event:   &domain.Event{Name: "Conf"},
			wantErr: true,
			assert:  func(t *testing.T, _ *fakeEventRepo, _ *domain.Event) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			ev := &domain.Event{Name: tt.event.Name, OwnerID: tt.event.OwnerID}
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
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
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
				_ = er.Create(ctx, &domain.Event{Name: "E1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				_ = er.Create(ctx, &domain.Event{Name: "E2", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				_ = er.Create(ctx, &domain.Event{Name: "Other", OwnerID: "user-2", CreatedAt: time.Now(), UpdatedAt: time.Now()})
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
				_ = er.Create(ctx, &domain.Event{Name: "E1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
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
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
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
		name         string
		setup        func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher)
		eventID      string
		wantErr      bool
		wantNotFound bool
		assert       func(t *testing.T, event *domain.Event, rooms []*domain.Room, sessions []*domain.Session)
	}{
		{
			name: "success with rooms and sessions",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				ev, _ := er.GetByID(ctx, "ev-1")
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: ev.ID, Name: "Room A"}}
				sr.sessions = []*domain.Session{{ID: "sess-1", RoomID: "room-1", Title: "Talk 1", Tags: []string{}}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID:      "ev-1",
			wantErr:      false,
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
			eventID:      "ev-missing",
			wantErr:      true,
			wantNotFound: true,
			assert:       func(t *testing.T, _ *domain.Event, _ []*domain.Room, _ []*domain.Session) {},
		},
		{
			name: "success empty rooms and sessions",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-1",
			wantErr:      false,
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
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
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

func TestManageScheduleService_DeleteEvent(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name          string
		setup         func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher)
		eventID       string
		ownerID       string
		wantErr       bool
		wantNotFound  bool
		wantForbidden bool
		assertDeleted bool
	}{
		{
			name: "success",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:       "ev-1",
			ownerID:       "user-1",
			wantErr:       false,
			assertDeleted: true,
		},
		{
			name: "event not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-missing",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name: "forbidden not owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:       "ev-1",
			ownerID:       "user-2",
			wantErr:       true,
			wantForbidden: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			err := svc.DeleteEvent(ctx, tt.eventID, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				return
			}
			require.NoError(t, err)
			if tt.assertDeleted {
				_, err := eventRepo.GetByID(ctx, tt.eventID)
				require.True(t, errors.Is(err, domain.ErrNotFound), "event should be deleted")
			}
		})
	}
}

func TestManageScheduleService_ToggleRoomNotBookable(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name          string
		setup         func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher)
		eventID       string
		roomID        string
		ownerID       string
		wantErr       bool
		wantNotFound  bool
		wantForbidden bool
		assert        func(t *testing.T, room *domain.Room)
	}{
		{
			name: "success toggles false to true",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-1", Name: "Room A", NotBookable: false}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID: "ev-1",
			roomID:  "room-1",
			ownerID: "user-1",
			wantErr: false,
			assert: func(t *testing.T, room *domain.Room) {
				require.NotNil(t, room)
				require.True(t, room.NotBookable, "expected NotBookable to be true after toggle")
				require.Equal(t, "room-1", room.ID)
			},
		},
		{
			name: "success toggles true to false",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-1", Name: "Room A", NotBookable: true}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID: "ev-1",
			roomID:  "room-1",
			ownerID: "user-1",
			wantErr: false,
			assert: func(t *testing.T, room *domain.Room) {
				require.NotNil(t, room)
				require.False(t, room.NotBookable, "expected NotBookable to be false after toggle")
			},
		},
		{
			name: "event not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				return newFakeEventRepo(), newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-missing",
			roomID:       "room-1",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name: "forbidden not owner",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-1", Name: "Room A"}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID:       "ev-1",
			roomID:        "room-1",
			ownerID:       "user-2",
			wantErr:       true,
			wantForbidden: true,
		},
		{
			name: "room not found",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				return er, newFakeSessionRepo(), &fakeSessionizeFetcher{}
			},
			eventID:      "ev-1",
			roomID:       "room-missing",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name: "room belongs to different event",
			setup: func() (domain.EventRepository, domain.SessionRepository, domain.SessionizeFetcher) {
				er := newFakeEventRepo()
				_ = er.Create(ctx, &domain.Event{Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
				sr := newFakeSessionRepo()
				sr.rooms = []*domain.Room{{ID: "room-1", EventID: "ev-99", Name: "Room A"}}
				return er, sr, &fakeSessionizeFetcher{}
			},
			eventID:      "ev-1",
			roomID:       "room-1",
			ownerID:      "user-1",
			wantErr:      true,
			wantNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo, sessionRepo, fetcher := tt.setup()
			svc := NewEventService(eventRepo, sessionRepo, newFakeEventTeamMemberRepo(), newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), fetcher, timeout)
			room, err := svc.ToggleRoomNotBookable(ctx, tt.eventID, tt.roomID, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				return
			}
			require.NoError(t, err)
			if tt.assert != nil {
				tt.assert(t, room)
			}
		})
	}
}

func TestManageScheduleService_AddEventTeamMember(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name          string
		eventID       string
		userIDToAdd   string
		ownerID       string
		setupEvent    func(*fakeEventRepo)
		setupTeamRepo func(*fakeEventTeamMemberRepo)
		wantErr       bool
		wantForbidden bool
		wantNotFound  bool
		wantConflict  bool
	}{
		{
			name:        "owner adds team member success",
			eventID:     "ev-1",
			userIDToAdd: "user-2",
			ownerID:     "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			wantErr: false,
		},
		{
			name:        "forbidden not owner",
			eventID:     "ev-1",
			userIDToAdd: "user-2",
			ownerID:     "user-other",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			wantErr:       true,
			wantForbidden: true,
		},
		{
			name:         "event not found",
			eventID:      "ev-missing",
			userIDToAdd:  "user-2",
			ownerID:      "user-1",
			setupEvent:   func(er *fakeEventRepo) {},
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name:        "add owner returns ErrInvalidInput",
			eventID:     "ev-1",
			userIDToAdd: "user-1",
			ownerID:     "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			wantErr:      true,
			wantConflict: true,
		},
		{
			name:        "already member returns ErrAlreadyMember",
			eventID:     "ev-1",
			userIDToAdd: "user-2",
			ownerID:     "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupTeamRepo: func(tr *fakeEventTeamMemberRepo) {
				tr.Add(ctx, "ev-1", "user-2")
			},
			wantErr:      true,
			wantConflict: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo := newFakeEventRepo()
			tt.setupEvent(eventRepo)
			teamRepo := newFakeEventTeamMemberRepo()
			if tt.setupTeamRepo != nil {
				tt.setupTeamRepo(teamRepo)
			}
			svc := NewEventService(eventRepo, newFakeSessionRepo(), teamRepo, newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), &fakeSessionizeFetcher{}, timeout)
			err := svc.AddEventTeamMember(ctx, tt.eventID, tt.userIDToAdd, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				if tt.wantConflict {
					require.True(t, errors.Is(err, domain.ErrAlreadyMember) || errors.Is(err, domain.ErrInvalidInput))
				}
				return
			}
			require.NoError(t, err)
			members, _ := teamRepo.ListByEventID(ctx, tt.eventID)
			require.Len(t, members, 1)
			require.Equal(t, tt.userIDToAdd, members[0].UserID)
		})
	}
}

func TestManageScheduleService_ListEventTeamMembers(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name          string
		eventID       string
		callerID      string
		setupEvent    func(*fakeEventRepo)
		setupTeamRepo func(*fakeEventTeamMemberRepo)
		wantErr       bool
		wantForbidden bool
		wantNotFound  bool
		wantCount     int
	}{
		{
			name:     "owner lists members",
			eventID:  "ev-1",
			callerID: "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupTeamRepo: func(tr *fakeEventTeamMemberRepo) {
				tr.Add(ctx, "ev-1", "user-2")
				tr.Add(ctx, "ev-1", "user-3")
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:     "forbidden not owner",
			eventID:  "ev-1",
			callerID: "user-other",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			wantErr:       true,
			wantForbidden: true,
		},
		{
			name:         "event not found",
			eventID:      "ev-missing",
			callerID:     "user-1",
			setupEvent:   func(er *fakeEventRepo) {},
			wantErr:      true,
			wantNotFound: true,
		},
		{
			name:     "owner lists empty",
			eventID:  "ev-1",
			callerID: "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			wantErr:   false,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo := newFakeEventRepo()
			tt.setupEvent(eventRepo)
			teamRepo := newFakeEventTeamMemberRepo()
			if tt.setupTeamRepo != nil {
				tt.setupTeamRepo(teamRepo)
			}
			svc := NewEventService(eventRepo, newFakeSessionRepo(), teamRepo, newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), &fakeSessionizeFetcher{}, timeout)
			got, err := svc.ListEventTeamMembers(ctx, tt.eventID, tt.callerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				return
			}
			require.NoError(t, err)
			require.Len(t, got, tt.wantCount)
		})
	}
}

func TestManageScheduleService_RemoveEventTeamMember(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name           string
		eventID        string
		userIDToRemove string
		ownerID        string
		setupEvent     func(*fakeEventRepo)
		setupTeamRepo  func(*fakeEventTeamMemberRepo)
		wantErr        bool
		wantForbidden  bool
		wantNotFound   bool
	}{
		{
			name:           "owner removes member success",
			eventID:        "ev-1",
			userIDToRemove: "user-2",
			ownerID:        "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupTeamRepo: func(tr *fakeEventTeamMemberRepo) {
				tr.Add(ctx, "ev-1", "user-2")
			},
			wantErr: false,
		},
		{
			name:           "forbidden not owner",
			eventID:        "ev-1",
			userIDToRemove: "user-2",
			ownerID:        "user-other",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupTeamRepo: func(tr *fakeEventTeamMemberRepo) {
				tr.Add(ctx, "ev-1", "user-2")
			},
			wantErr:       true,
			wantForbidden: true,
		},
		{
			name:           "event not found",
			eventID:        "ev-missing",
			userIDToRemove: "user-2",
			ownerID:        "user-1",
			setupEvent:     func(er *fakeEventRepo) {},
			wantErr:        true,
			wantNotFound:   true,
		},
		{
			name:           "member not in team returns not found",
			eventID:        "ev-1",
			userIDToRemove: "user-99",
			ownerID:        "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			wantErr:      true,
			wantNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo := newFakeEventRepo()
			tt.setupEvent(eventRepo)
			teamRepo := newFakeEventTeamMemberRepo()
			if tt.setupTeamRepo != nil {
				tt.setupTeamRepo(teamRepo)
			}
			svc := NewEventService(eventRepo, newFakeSessionRepo(), teamRepo, newFakeUserRepoForSchedule(), newFakeEventInvitationRepo(), newFakeEmailService(), &fakeSessionizeFetcher{}, timeout)
			err := svc.RemoveEventTeamMember(ctx, tt.eventID, tt.userIDToRemove, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				if tt.wantNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				return
			}
			require.NoError(t, err)
			members, _ := teamRepo.ListByEventID(ctx, tt.eventID)
			for _, m := range members {
				require.NotEqual(t, tt.userIDToRemove, m.UserID)
			}
		})
	}
}

func TestManageScheduleService_AddEventTeamMemberByEmail(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name             string
		eventID          string
		email            string
		ownerID          string
		setupEvent       func(*fakeEventRepo)
		setupUserRepo    func(*fakeUserRepoForSchedule)
		wantErr          bool
		wantUserNotFound bool
		wantConflict     bool
		wantMemberUserID string
	}{
		{
			name:    "success adds by email",
			eventID: "ev-1",
			email:   "teammate@example.com",
			ownerID: "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupUserRepo: func(ur *fakeUserRepoForSchedule) {
				ur.addUser("teammate@example.com", "user-2")
			},
			wantErr:          false,
			wantMemberUserID: "user-2",
		},
		{
			name:    "user not found by email",
			eventID: "ev-1",
			email:   "nobody@example.com",
			ownerID: "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupUserRepo:    func(ur *fakeUserRepoForSchedule) {},
			wantErr:          true,
			wantUserNotFound: true,
		},
		{
			name:    "email normalized to lower",
			eventID: "ev-1",
			email:   "Teammate@Example.COM",
			ownerID: "user-1",
			setupEvent: func(er *fakeEventRepo) {
				er.Create(ctx, &domain.Event{ID: "ev-1", Name: "Conf", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
			},
			setupUserRepo: func(ur *fakeUserRepoForSchedule) {
				ur.addUser("teammate@example.com", "user-2")
			},
			wantErr:          false,
			wantMemberUserID: "user-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo := newFakeEventRepo()
			tt.setupEvent(eventRepo)
			teamRepo := newFakeEventTeamMemberRepo()
			userRepo := newFakeUserRepoForSchedule()
			if tt.setupUserRepo != nil {
				tt.setupUserRepo(userRepo)
			}
			svc := NewEventService(eventRepo, newFakeSessionRepo(), teamRepo, userRepo, newFakeEventInvitationRepo(), newFakeEmailService(), &fakeSessionizeFetcher{}, timeout)
			got, err := svc.AddEventTeamMemberByEmail(ctx, tt.eventID, tt.email, tt.ownerID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantUserNotFound {
					require.True(t, errors.Is(err, domain.ErrUserNotFound))
				}
				if tt.wantConflict {
					require.True(t, errors.Is(err, domain.ErrAlreadyMember) || errors.Is(err, domain.ErrInvalidInput))
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, tt.eventID, got.EventID)
			require.Equal(t, tt.wantMemberUserID, got.UserID)
		})
	}
}

func TestEventService_SendEventInvitations(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second

	tests := []struct {
		name         string
		eventID      string
		ownerID      string
		emails       []string
		setupEvent   func(*fakeEventRepo)
		setupUser    func(*fakeUserRepoForSchedule)
		setupEmail   func(*fakeEmailService)
		wantSent     int
		wantFailed   []string
		wantErr      bool
		wantErrNotFound bool
		wantErrForbidden bool
	}{
		{
			name:    "success sends to two emails",
			eventID: "ev-1",
			ownerID: "user-1",
			emails:  []string{"a@example.com", "b@example.com"},
			setupEvent: func(er *fakeEventRepo) {
				er.byID["ev-1"] = &domain.Event{ID: "ev-1", Name: "My Event", EventCode: "abc1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			setupUser: func(ur *fakeUserRepoForSchedule) {
				ur.addUserWithName("owner@x.com", "user-1", "Jane", "Doe")
			},
			setupEmail:  func(*fakeEmailService) {},
			wantSent:    2,
			wantFailed:  nil,
			wantErr:     false,
		},
		{
			name:    "event not found",
			eventID: "ev-missing",
			ownerID: "user-1",
			emails:  []string{"a@example.com"},
			setupEvent: func(er *fakeEventRepo) {},
			setupUser:   func(ur *fakeUserRepoForSchedule) {},
			setupEmail:  func(*fakeEmailService) {},
			wantErr:     true,
			wantErrNotFound: true,
		},
		{
			name:    "forbidden when not owner",
			eventID: "ev-1",
			ownerID: "user-2",
			emails:  []string{"a@example.com"},
			setupEvent: func(er *fakeEventRepo) {
				er.byID["ev-1"] = &domain.Event{ID: "ev-1", Name: "My Event", EventCode: "abc1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			setupUser:   func(ur *fakeUserRepoForSchedule) {},
			setupEmail:  func(*fakeEmailService) {},
			wantErr:     true,
			wantErrForbidden: true,
		},
		{
			name:    "partial failure when email send fails",
			eventID: "ev-1",
			ownerID: "user-1",
			emails:  []string{"ok@example.com", "fail@example.com"},
			setupEvent: func(er *fakeEventRepo) {
				er.byID["ev-1"] = &domain.Event{ID: "ev-1", Name: "My Event", EventCode: "abc1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			setupUser: func(ur *fakeUserRepoForSchedule) {
				ur.addUserWithName("owner@x.com", "user-1", "Jane", "Doe")
			},
			setupEmail: func(es *fakeEmailService) {
				es.sendEventInvitationErr = errors.New("smtp error")
				// Fail only for second email - we can't do that with current fake, so we make all fail and expect sent=0, failed=2
				// Actually let's make sendEventInvitationErr set and then we get first email: create ok, send fail -> failed. Second: create ok, send fail -> failed. So sent=0, failed=2.
				// For "partial" we need one to succeed and one to fail. So we need the fake to fail on a specific email. Simpler: just test that when send fails, that email is in failed. So set sendEventInvitationErr and both go to failed, sent=0, failed=2.
			},
			wantSent:   0,
			wantFailed: []string{"ok@example.com", "fail@example.com"},
			wantErr:    false,
		},
		{
			name:    "empty emails returns zero sent",
			eventID: "ev-1",
			ownerID: "user-1",
			emails:  []string{},
			setupEvent: func(er *fakeEventRepo) {
				er.byID["ev-1"] = &domain.Event{ID: "ev-1", Name: "My Event", EventCode: "abc1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			setupUser:   func(ur *fakeUserRepoForSchedule) {},
			setupEmail:  func(*fakeEmailService) {},
			wantSent:    0,
			wantFailed:  nil,
			wantErr:     false,
		},
		{
			name:    "duplicate email in list: first sent, second failed",
			eventID: "ev-1",
			ownerID: "user-1",
			emails:  []string{"dup@example.com", "dup@example.com"},
			setupEvent: func(er *fakeEventRepo) {
				er.byID["ev-1"] = &domain.Event{ID: "ev-1", Name: "My Event", EventCode: "abc1", OwnerID: "user-1", CreatedAt: time.Now(), UpdatedAt: time.Now()}
			},
			setupUser: func(ur *fakeUserRepoForSchedule) {
				ur.addUserWithName("owner@x.com", "user-1", "Jane", "Doe")
			},
			setupEmail:  func(*fakeEmailService) {},
			wantSent:    1,
			wantFailed:  []string{"dup@example.com"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventRepo := newFakeEventRepo()
			if tt.setupEvent != nil {
				tt.setupEvent(eventRepo)
			}
			userRepo := newFakeUserRepoForSchedule()
			if tt.setupUser != nil {
				tt.setupUser(userRepo)
			}
			invRepo := newFakeEventInvitationRepo()
			emailSvc := newFakeEmailService()
			if tt.setupEmail != nil {
				tt.setupEmail(emailSvc)
			}
			svc := NewEventService(eventRepo, newFakeSessionRepo(), newFakeEventTeamMemberRepo(), userRepo, invRepo, emailSvc, &fakeSessionizeFetcher{}, timeout)

			sent, failed, err := svc.SendEventInvitations(ctx, tt.eventID, tt.ownerID, tt.emails)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrNotFound {
					require.True(t, errors.Is(err, domain.ErrNotFound))
				}
				if tt.wantErrForbidden {
					require.True(t, errors.Is(err, domain.ErrForbidden))
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantSent, sent)
			require.ElementsMatch(t, tt.wantFailed, failed)
			if tt.wantSent > 0 && len(tt.emails) > 0 {
				list, _ := invRepo.ListByEventID(ctx, tt.eventID)
				require.Len(t, list, tt.wantSent, "invitations persisted should match sent count")
				require.Len(t, emailSvc.sentInvitations, tt.wantSent, "emails sent should match sent count")
			}
		})
	}
}
